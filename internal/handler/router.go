package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/client"
	"elayan.local/desa/internal/domain"
	"elayan.local/desa/internal/gemini"
	"elayan.local/desa/internal/id"
	"elayan.local/desa/internal/messenger"
	"elayan.local/desa/internal/repository"
	"elayan.local/desa/internal/storage"
	"elayan.local/desa/pkg/utils"
	"elayan.local/desa/pkg/validator"
)

type Router struct {
	db            *sql.DB
	wa            messenger.Messenger
	api           *client.Backend
	files         *storage.Local
	retentionDays int
	logger        *log.Logger
	ai            *gemini.Assistant // nil/nonaktif = hanya FAQ kata kunci
	mu            sync.Mutex
}

type conversation struct {
	RequestID    string              `json:"requestId"`
	LetterCode   string              `json:"letterCode"`
	Requirements []string            `json:"requirements,omitempty"`
	Summary      string              `json:"summary"`
	Attachments  []client.Attachment `json:"attachments"`
}

func NewRouter(db *sql.DB, wa messenger.Messenger, api *client.Backend, files *storage.Local, retentionDays int, logger *log.Logger, ai *gemini.Assistant) *Router {
	return &Router{db: db, wa: wa, api: api, files: files, retentionDays: retentionDays, logger: logger, ai: ai}
}

func (r *Router) Start(ctx context.Context) {
	r.wa.OnMessage(func(message messenger.Message) {
		go r.Handle(ctx, message)
	})
	go r.expireSessions(ctx)
}

func (r *Router) Handle(parent context.Context, message messenger.Message) {
	if message.JID == "" || message.PhoneJID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	r.mu.Lock()
	locked := true
	defer func() {
		if locked {
			r.mu.Unlock()
		}
	}()
	if ctx.Err() != nil {
		return
	}

	session, err := repository.LoadBotSession(ctx, r.db, message.JID)
	state := conversation{}
	if err == nil {
		if time.Since(session.DiubahPada) > 24*time.Hour || json.Unmarshal(session.Data, &state) != nil {
			r.removeStaged(state)
			_ = repository.DeleteBotSession(ctx, r.db, message.JID)
			state = conversation{}
			session = domain.SesiBot{}
		}
	} else if err != sql.ErrNoRows {
		r.reply(ctx, message.JID, "Layanan sedang bermasalah. Coba lagi sebentar.")
		return
	}

	text := strings.TrimSpace(message.Text)
	upper := strings.ToUpper(text)
	if upper == "BATAL" {
		r.removeStaged(state)
		_ = repository.DeleteBotSession(ctx, r.db, message.JID)
		r.reply(ctx, message.JID, "Pengajuan dibatalkan. Ketik MENU untuk mulai lagi.")
		return
	}
	if code := utils.RequestCode(text); code != "" {
		r.checkStatus(ctx, message, code)
		return
	}
	// Sapaan ditangani kode biasa (tanpa Gemini): cepat, tidak memakai kuota,
	// dan tidak mengirim pesan warga ke layanan luar.
	if message.Media == nil && isGreeting(text) {
		r.reply(ctx, message.JID, menuText)
		return
	}
	if upper == "SURAT" {
		r.removeStaged(state)
		requestID, err := id.UUID()
		if err != nil {
			r.reply(ctx, message.JID, "Pengajuan belum dapat dimulai. Coba lagi sebentar.")
			return
		}
		state = conversation{RequestID: requestID}
		session = domain.SesiBot{JID: message.JID, Flow: "surat", Step: "jenis"}
		if err := r.save(ctx, session, state); err != nil {
			r.reply(ctx, message.JID, "Pengajuan belum dapat dimulai. Coba lagi sebentar.")
			return
		}
		r.showLetterTypes(ctx, message.JID)
		return
	}

	if message.Media != nil {
		r.handlePhoto(ctx, message, session, state, text)
		return
	}
	if session.Flow == "surat" {
		// Pertanyaan bantuan di tengah alur dijawab tanpa mengubah langkah
		// permohonan. Gemini hanya menerima teks pertanyaan yang lolos filter PII;
		// state, ringkasan, lampiran, dan data permohonan tidak ikut dikirim.
		if isHelpQuestion(text) {
			r.mu.Unlock()
			locked = false
			r.answerFAQ(ctx, message.JID, text)
			return
		}
		r.handleLetterFlow(ctx, message, session, state, text)
		return
	}
	// Di luar alur surat: identitas bot dan ucapan terima kasih dijawab
	// kode biasa (tanpa AI) agar konsisten, cepat, dan tidak memakai kuota.
	if isIdentityQuestion(text) {
		r.reply(ctx, message.JID, identityText)
		return
	}
	if isThanks(text) {
		r.reply(ctx, message.JID, thanksText)
		return
	}
	// Jawaban FAQ/AI tidak mengubah sesi, jadi lepas kunci global agar
	// panggilan Gemini (bisa belasan detik) tidak memblokir warga lain.
	r.mu.Unlock()
	locked = false
	r.answerFAQ(ctx, message.JID, strings.TrimPrefix(text, "INFO "))
}

func (r *Router) showLetterTypes(ctx context.Context, jid string) {
	types, err := r.api.LetterTypes(ctx)
	if err != nil {
		r.reply(ctx, jid, "Daftar surat belum dapat dimuat. Coba lagi sebentar atau ketik BATAL.")
		return
	}
	var menu strings.Builder
	menu.WriteString("Pilih jenis surat dengan mengetik nomor atau kodenya. Contoh: ketik 1 atau DM untuk memilih Surat Keterangan Domisili.\n\n")
	for i, item := range types {
		fmt.Fprintf(&menu, "*%d. %s (%s)*\n", i+1, item.Name, item.Code)
		if len(item.Requirements) > 0 {
			fmt.Fprintf(&menu, "   Syarat: %s\n", strings.Join(item.Requirements, ", "))
		}
		menu.WriteByte('\n')
	}
	menu.WriteString("Syarat resmi dapat berbeda menurut jenis surat; bot menerima foto KTP, KK, atau pengantar RT sesuai arahan perangkat desa.")
	r.reply(ctx, jid, menu.String())
}

func (r *Router) handleLetterFlow(ctx context.Context, message messenger.Message, session domain.SesiBot, state conversation, text string) {
	if state.LetterCode != "" && len(state.Requirements) == 0 {
		types, err := r.api.LetterTypes(ctx)
		if err != nil {
			r.reply(ctx, message.JID, "Syarat dokumen belum dapat dimuat. Coba lagi sebentar.")
			return
		}
		for _, item := range types {
			if item.Code == state.LetterCode {
				state.Requirements = item.Requirements
				break
			}
		}
		if len(state.Requirements) == 0 {
			r.reply(ctx, message.JID, "Jenis surat tidak tersedia lagi. Ketik SURAT untuk memilih jenis lain.")
			return
		}
		if err := r.save(ctx, session, state); err != nil {
			r.reply(ctx, message.JID, "Syarat dokumen belum tersimpan. Coba lagi sebentar.")
			return
		}
	}
	switch session.Step {
	case "jenis":
		types, err := r.api.LetterTypes(ctx)
		if err != nil {
			r.reply(ctx, message.JID, "Daftar surat belum dapat dimuat. Coba lagi sebentar.")
			return
		}
		selection := strings.ToUpper(strings.TrimSpace(text))
		for i, item := range types {
			if selection == item.Code || selection == fmt.Sprint(i+1) {
				state.LetterCode = item.Code
				state.Requirements = item.Requirements
				session.Step = "keperluan"
				if err := r.save(ctx, session, state); err != nil {
					r.reply(ctx, message.JID, "Pilihan belum tersimpan. Coba lagi sebentar.")
					return
				}
				r.reply(ctx, message.JID, fmt.Sprintf("Syarat dokumen %s: %s.\nJelaskan singkat keperluan surat ini.", item.Name, strings.Join(item.Requirements, ", ")))
				return
			}
		}
		r.showLetterTypes(ctx, message.JID)
	case "keperluan":
		if utf8.RuneCountInString(text) < 1 || utf8.RuneCountInString(text) > 500 {
			r.reply(ctx, message.JID, "Keperluan harus berisi 1 sampai 500 karakter. Silakan kirim ulang.")
			return
		}
		state.Summary = text
		session.Step = "persetujuan"
		if err := r.save(ctx, session, state); err != nil {
			r.reply(ctx, message.JID, "Data belum tersimpan. Coba lagi sebentar.")
			return
		}
		r.reply(ctx, message.JID, fmt.Sprintf("Dokumen yang perlu disiapkan: %s. Foto dokumen akan diproses untuk pengajuan surat, dilihat petugas desa melalui dashboard, dan dihapus %d hari setelah surat selesai. Ketik SETUJU untuk melanjutkan.", strings.Join(state.Requirements, ", "), r.retentionDays))
	case "persetujuan":
		if strings.ToUpper(strings.TrimSpace(text)) != "SETUJU" {
			r.reply(ctx, message.JID, "Untuk melanjutkan, ketik SETUJU. Ketik BATAL untuk berhenti.")
			return
		}
		session.Step = "lampiran"
		if err := r.save(ctx, session, state); err != nil {
			r.reply(ctx, message.JID, "Persetujuan belum tersimpan. Coba lagi sebentar.")
			return
		}
		r.reply(ctx, message.JID, fmt.Sprintf("Kirim foto sesuai syarat: %s. Gunakan caption KTP, KK, RT, atau LAINNYA pada tiap foto. Setelah semua foto terkirim, ketik SELESAI. Maksimal 5 foto, JPEG/PNG, 10 MB per foto.", strings.Join(state.Requirements, ", ")))
	case "lampiran":
		if strings.ToUpper(strings.TrimSpace(text)) == "SELESAI" {
			r.submit(ctx, message, state)
			return
		}
		r.reply(ctx, message.JID, "Kirim foto dengan caption KTP, KK, RT, atau LAINNYA; ketik SELESAI jika sudah lengkap.")
	default:
		r.reply(ctx, message.JID, "Sesi pengajuan tidak valid. Ketik SURAT untuk memulai lagi.")
	}
}

func (r *Router) handlePhoto(ctx context.Context, message messenger.Message, session domain.SesiBot, state conversation, caption string) {
	if session.Flow != "surat" || session.Step != "lampiran" {
		r.reply(ctx, message.JID, "Untuk mengirim foto, mulai pengajuan dengan mengetik SURAT terlebih dahulu.")
		return
	}
	kind := strings.ToUpper(strings.TrimSpace(caption))
	if kind != "KTP" && kind != "KK" && kind != "RT" && kind != "LAINNYA" {
		r.reply(ctx, message.JID, "Foto belum diterima. Sertakan caption KTP, KK, RT, atau LAINNYA pada foto.")
		return
	}
	if len(state.Attachments) >= 5 {
		r.reply(ctx, message.JID, "Batas 5 foto sudah tercapai. Ketik SELESAI untuk mengirim pengajuan.")
		return
	}
	if message.Media.Size == 0 || message.Media.Size > 10<<20 || message.Media.MIME != "image/jpeg" && message.Media.MIME != "image/png" {
		r.reply(ctx, message.JID, "Foto harus JPEG atau PNG dan maksimal 10 MB.")
		return
	}
	data, err := r.wa.DownloadMedia(ctx, *message.Media)
	if err != nil {
		r.reply(ctx, message.JID, "Foto belum dapat diunduh. Coba kirim ulang sebagai foto JPEG atau PNG.")
		return
	}
	mime, extension, err := validator.ImageType(data, 10<<20)
	if err != nil || mime != message.Media.MIME {
		r.reply(ctx, message.JID, "Lampiran harus foto JPEG atau PNG yang valid dan maksimal 10 MB.")
		return
	}
	storageID, err := id.UUID()
	if err != nil {
		r.reply(ctx, message.JID, "Foto belum dapat disimpan. Coba lagi sebentar.")
		return
	}
	key := storageID + extension
	if err := r.files.Create(key, data); err != nil {
		r.reply(ctx, message.JID, "Foto belum dapat disimpan. Coba lagi sebentar.")
		return
	}
	state.Attachments = append(state.Attachments, client.Attachment{Key: key, Kind: kind, MIME: mime, Size: int64(len(data))})
	if err := r.save(ctx, session, state); err != nil {
		_ = r.files.Remove(key)
		r.reply(ctx, message.JID, "Foto belum tercatat. Silakan kirim ulang.")
		return
	}
	r.reply(ctx, message.JID, fmt.Sprintf("Foto %d diterima. Kirim foto berikutnya atau ketik SELESAI.", len(state.Attachments)))
}

func (r *Router) submit(ctx context.Context, message messenger.Message, state conversation) {
	if len(state.Attachments) == 0 {
		r.reply(ctx, message.JID, "Kirim minimal satu foto persyaratan sebelum mengetik SELESAI.")
		return
	}
	result, err := r.api.CreateSubmission(ctx, client.Submission{
		RequestID: state.RequestID, LetterCode: state.LetterCode, WAJID: message.PhoneJID,
		Summary: state.Summary, ConsentVersion: "v1", Attachments: state.Attachments,
	})
	if err != nil {
		r.reply(ctx, message.JID, "Pengajuan belum mendapat konfirmasi dari server. Data sementara tersimpan 24 jam; ketik SELESAI lagi untuk mencoba ulang atau BATAL untuk membatalkan.")
		return
	}
	if err := r.reply(ctx, message.JID, fmt.Sprintf("Pengajuan diterima. Simpan kode %s untuk mengecek status kapan saja. Status saat ini: %s.", result.Code, result.Status)); err != nil {
		return
	}
	r.removeStaged(state)
	if err := repository.DeleteBotSession(ctx, r.db, message.JID); err != nil {
		r.logger.Printf("bot session cleanup failed: %v", err)
	}
}

func (r *Router) checkStatus(ctx context.Context, message messenger.Message, code string) {
	result, err := r.api.CheckStatus(ctx, code, message.PhoneJID)
	if errors.Is(err, client.ErrNotFound) {
		// Pesan sama untuk "kode tidak ada" dan "kode milik nomor lain",
		// agar orang luar tidak bisa menebak kode permohonan warga.
		r.reply(ctx, message.JID, statusNotFoundText)
		return
	}
	if err != nil {
		r.reply(ctx, message.JID, "Status belum dapat diperiksa. Coba lagi sebentar.")
		return
	}
	r.reply(ctx, message.JID, fmt.Sprintf("Status pengajuan %s: %s.", result.Code, result.Status))
}

func (r *Router) answerFAQ(ctx context.Context, jid, query string) {
	query = strings.TrimSpace(query)
	if strings.EqualFold(query, "INFO") || query == "" {
		r.reply(ctx, jid, "Kirim pertanyaan tentang jam layanan, persyaratan, alur pengurusan, atau kontak desa.")
		return
	}
	// Gemini diprioritaskan agar bisa mencocokkan maksud dan bahasa santai.
	// Model hanya diberi FAQ aktif; ia tidak mendapat state atau data permohonan.
	if r.ai.Enabled() {
		entries, err := repository.ListActiveFAQ(ctx, r.db, 100)
		if err != nil {
			r.logger.Printf("faq load for ai failed: %v", err)
			r.reply(ctx, jid, "Informasi belum dapat dimuat. Coba lagi sebentar.")
			return
		}
		facts := make([]gemini.Fact, 0, len(entries)+1)
		facts = append(facts, identityFact)
		for _, e := range entries {
			facts = append(facts, gemini.Fact{Question: e.Question, Answer: e.Answer})
		}
		r.reply(ctx, jid, r.ai.Answer(ctx, jid, query, facts))
		return
	}
	// Tanpa Gemini, pencocokan FAQ deterministik tetap tersedia.
	answer, err := r.api.FAQ(ctx, query)
	if err == nil {
		r.reply(ctx, jid, answer)
		return
	}
	if !errors.Is(err, client.ErrNotFound) {
		r.reply(ctx, jid, "Informasi belum dapat dimuat. Coba lagi sebentar.")
		return
	}
	if !r.ai.Enabled() {
		r.reply(ctx, jid, "Informasi itu belum tersedia di layanan ini. Bot hanya menjawab informasi desa yang sudah disiapkan perangkat desa. Ketik MENU untuk melihat layanan.")
		return
	}
}

func isHelpQuestion(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	if strings.Contains(text, "?") {
		return true
	}
	for _, prefix := range []string{
		"apa ", "apa itu", "bagaimana ", "gimana ", "kenapa ", "mengapa ",
		"syarat ", "berapa ", "boleh ", "bisakah ", "bisa ", "cara ",
		"maksud ", "arti ", "kalau ", "kalo ", "untuk apa", "tolong jelaskan",
		"mau tanya", "izin tanya", "nanya",
	} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func (r *Router) save(ctx context.Context, session domain.SesiBot, state conversation) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	session.Data = data
	return repository.SaveBotSession(ctx, r.db, session)
}

func (r *Router) removeStaged(state conversation) {
	for _, attachment := range state.Attachments {
		if err := r.files.Remove(attachment.Key); err != nil {
			r.logger.Printf("staged attachment cleanup failed: %v", err)
		}
	}
}

func (r *Router) reply(ctx context.Context, jid, text string) error {
	if err := r.wa.SendText(ctx, jid, text); err != nil {
		r.logger.Printf("WhatsApp reply failed: %v", err)
		return err
	}
	return nil
}

func (r *Router) expireSessions(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		expired, err := repository.ExpireBotSessions(ctx, r.db)
		if err != nil && ctx.Err() == nil {
			r.logger.Printf("bot session cleanup failed: %v", err)
		}
		for _, session := range expired {
			var state conversation
			if json.Unmarshal(session.Data, &state) == nil {
				r.removeStaged(state)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}