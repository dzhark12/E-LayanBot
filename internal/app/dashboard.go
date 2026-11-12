package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/domain"
	"elayan.local/desa/internal/pdf"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var nikPattern = regexp.MustCompile(`^[0-9]{16}$`)

type letterField struct {
	Key, Label, Type, Value string
}

var letterFields = map[string][]letterField{
	"DM": {
		{"NoUrut", "Nomor surat", "text", ""}, {"Nama", "Nama lengkap", "text", ""},
		{"JenisKelamin", "Jenis kelamin", "gender", ""}, {"TempatLahir", "Tempat lahir", "text", ""},
		{"TanggalLahir", "Tanggal lahir", "date", ""}, {"Nik", "NIK", "nik", ""},
		{"Agama", "Agama", "text", ""}, {"StatusPerkawinan", "Status perkawinan", "text", ""},
		{"Pekerjaan", "Pekerjaan", "text", ""}, {"Alamat", "Alamat", "textarea", ""},
		{"Kecamatan", "Kecamatan", "text", "Tanjung Bintang"}, {"AlamatDomisili", "Alamat domisili", "textarea", ""},
	},
	"SU": {
		{"NoUrut", "Nomor surat", "text", ""}, {"Nama", "Nama lengkap", "text", ""},
		{"JenisKelamin", "Jenis kelamin", "gender", ""}, {"TempatLahir", "Tempat lahir", "text", ""},
		{"TanggalLahir", "Tanggal lahir", "date", ""}, {"Nik", "NIK", "nik", ""},
		{"Agama", "Agama", "text", ""}, {"StatusPerkawinan", "Status perkawinan", "text", ""},
		{"Pekerjaan", "Pekerjaan", "text", ""}, {"Alamat", "Alamat", "textarea", ""},
		{"Kecamatan", "Kecamatan", "text", "Tanjung Bintang"}, {"JenisUsaha", "Jenis usaha", "text", ""},
		{"LokasiUsaha", "Lokasi usaha", "textarea", ""},
	},
	"TM": {
		{"NoUrut", "Nomor surat", "text", ""}, {"Nama", "Nama lengkap", "text", ""},
		{"JenisKelamin", "Jenis kelamin", "gender", ""}, {"TempatLahir", "Tempat lahir", "text", ""},
		{"TanggalLahir", "Tanggal lahir", "date", ""}, {"Nik", "NIK", "nik", ""},
		{"Agama", "Agama", "text", ""}, {"Pekerjaan", "Pekerjaan", "text", ""},
		{"Alamat", "Alamat", "textarea", ""}, {"Kecamatan", "Kecamatan", "text", "Tanjung Bintang"},
		{"Keperluan", "Keperluan", "text", ""}, {"AnakNama", "Nama anak", "text", ""},
		{"AnakJenisKelamin", "Jenis kelamin anak", "gender", ""},
		{"AnakTempatTanggalLahir", "Tempat dan tanggal lahir anak", "text", ""},
		{"AnakNik", "NIK anak", "nik", ""}, {"AnakAgama", "Agama anak", "text", ""},
		{"AnakPekerjaan", "Pekerjaan anak", "text", ""}, {"AnakAlamat", "Alamat anak", "textarea", ""},
		{"AnakKecamatan", "Kecamatan anak", "text", ""},
	},
	"KM": {
		{"NoUrut", "Nomor surat", "text", ""}, {"Nama", "Nama almarhum/almarhumah", "text", ""},
		{"Nik", "NIK almarhum/almarhumah", "nik", ""}, {"TempatLahir", "Tempat lahir", "text", ""},
		{"TanggalLahir", "Tanggal lahir", "date", ""}, {"JenisKelamin", "Jenis kelamin", "gender", ""},
		{"Alamat", "Alamat", "textarea", ""}, {"Kecamatan", "Kecamatan", "text", "Tanjung Bintang"},
		{"HariMeninggal", "Hari meninggal", "text", ""}, {"TanggalMeninggal", "Tanggal meninggal", "date", ""},
		{"Penyebab", "Penyebab kematian", "text", ""},
		{"PelaporNama", "Nama pelapor", "text", ""}, {"PelaporNik", "NIK pelapor", "nik", ""},
		{"PelaporHubungan", "Hubungan dengan almarhum/almarhumah", "text", ""},
	},
}

type dashboardPage struct {
	User         adminUser
	CSRF         string
	Requests     []domain.Permohonan
	Status       string
	Query        string
	Page         int
	Pages        int
	Total        int
	Menunggu       int
	Dikerjakan     int
	Selesai        int
	PerluPerbaikan int
	Ditolak        int
	// RequestCount = jumlah seluruh permohonan tanpa filter; dipakai untuk tab
	// "Semua" dan sebagai versi data untuk penyegaran otomatis dashboard.
	RequestCount int64
	Notice       string
}

type detailPage struct {
	User     adminUser
	CSRF     string
	Request  domain.Permohonan
	Statuses []domain.Status
	Fields   []letterField
	HasPDF   bool
	Notice   string
}

func formFields(code string, identity domain.Identitas) []letterField {
	fields := append([]letterField(nil), letterFields[code]...)
	if len(fields) == 0 {
		fields = []letterField{
			{"Nama", "Nama lengkap", "text", ""}, {"TempatLahir", "Tempat lahir", "text", ""},
			{"TanggalLahir", "Tanggal lahir", "date", ""}, {"JenisKelamin", "Jenis kelamin", "gender", ""},
			{"Agama", "Agama", "text", ""}, {"Nik", "NIK", "nik", ""},
			{"Pekerjaan", "Pekerjaan", "text", ""}, {"Alamat", "Alamat", "textarea", ""},
		}
	}
	for i := range fields {
		if value := identityValue(identity, fields[i].Key); value != "" {
			fields[i].Value = value
		}
	}
	return fields
}

func identityValue(identity domain.Identitas, key string) string {
	switch key {
	case "Nama":
		return identity.Nama
	case "JenisKelamin":
		return identity.JenisKelamin
	case "TempatLahir":
		return identity.TempatLahir
	case "TanggalLahir":
		return identity.TanggalLahir
	case "Nik":
		return identity.NIK
	case "Agama":
		return identity.Agama
	case "Pekerjaan":
		return identity.Pekerjaan
	case "Alamat":
		return identity.Alamat
	case "Kecamatan":
		if value := identity.Tambahan[key]; value != "" {
			return value
		}
		return "Tanjung Bintang"
	default:
		return identity.Tambahan[key]
	}
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	user := currentAdmin(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var page dashboardPage
	page.User, page.CSRF = user, user.CSRF
	page.Query = strings.TrimSpace(r.URL.Query().Get("q"))
	if len(page.Query) > 100 {
		page.Query = page.Query[:100]
	}
	page.Status = r.URL.Query().Get("status")
	if !validStatus(page.Status) {
		page.Status = ""
	}
	page.Page = pageInt(r.URL.Query().Get("page"))
	page.Notice = r.URL.Query().Get("notice")
	// Hitungan per status untuk tab filter; selalu tanpa filter pencarian agar
	// angka tiap tab tidak berubah saat petugas berpindah tab atau mencari.
	if err := s.statusCounts(ctx, &page); err != nil {
		http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	countQuery := `SELECT count(*) FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE ($1='' OR p.status=$1) AND ($2='' OR p.kode_unik ILIKE '%'||$2||'%' OR j.nama ILIKE '%'||$2||'%' OR p.ringkasan ILIKE '%'||$2||'%')`
	if err := s.db.QueryRowContext(ctx, countQuery, page.Status, page.Query).Scan(&page.Total); err != nil {
		http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	page.Pages = (page.Total + 24) / 25
	if page.Pages == 0 {
		page.Pages = 1
	}
	if page.Page > page.Pages {
		page.Page = page.Pages
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id::text, p.kode_unik, j.kode, j.nama, p.ringkasan, p.status, p.dibuat_pada, p.diubah_pada FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE ($1='' OR p.status=$1) AND ($2='' OR p.kode_unik ILIKE '%'||$2||'%' OR j.nama ILIKE '%'||$2||'%' OR p.ringkasan ILIKE '%'||$2||'%') ORDER BY p.dibuat_pada DESC LIMIT 25 OFFSET $3`, page.Status, page.Query, (page.Page-1)*25)
	if err != nil {
		http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()
	page.Requests = make([]domain.Permohonan, 0, 25)
	for rows.Next() {
		var request domain.Permohonan
		var status string
		if err := rows.Scan(&request.ID, &request.Kode, &request.JenisKode, &request.JenisNama, &request.Ringkasan, &status, &request.DibuatPada, &request.DiubahPada); err != nil {
			http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		request.Status = domain.Status(status)
		page.Requests = append(page.Requests, request)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	rows.Close()
	s.render(w, "dashboard.html", page)
}

// statusCounts mengisi jumlah permohonan per status dan totalnya dengan satu query.
func (s *Server) statusCounts(ctx context.Context, page *dashboardPage) error {
	rows, err := s.db.QueryContext(ctx, `SELECT status, count(*) FROM permohonan GROUP BY status`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		page.RequestCount += int64(count)
		switch domain.Status(status) {
		case domain.Menunggu:
			page.Menunggu = count
		case domain.Dikerjakan:
			page.Dikerjakan = count
		case domain.Selesai:
			page.Selesai = count
		case domain.PerluPerbaikan:
			page.PerluPerbaikan = count
		case domain.Ditolak:
			page.Ditolak = count
		}
	}
	return rows.Err()
}

func (s *Server) requestVersion(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM permohonan`).Scan(&count); err != nil {
		http.Error(w, "Data permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]int64{"version": count})
}

func (s *Server) requestDetail(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var page detailPage
	page.User, page.CSRF, page.Statuses = currentAdmin(r), currentAdmin(r).CSRF, domain.Statuses
	page.Notice = r.URL.Query().Get("notice")
	var status string
	var identity []byte
	err := s.db.QueryRowContext(ctx, `SELECT p.id::text, p.kode_unik, p.wa_jid, j.kode, j.nama, p.ringkasan, p.status, p.data_identitas, p.dibuat_pada, p.diubah_pada FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1`, r.PathValue("id")).Scan(&page.Request.ID, &page.Request.Kode, &page.Request.WAJID, &page.Request.JenisKode, &page.Request.JenisNama, &page.Request.Ringkasan, &status, &identity, &page.Request.DibuatPada, &page.Request.DiubahPada)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil || json.Unmarshal(identity, &page.Request.Identitas) != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	page.Request.Status = domain.Status(status)
	page.Fields = formFields(page.Request.JenisKode, page.Request.Identitas)
	page.HasPDF = pdf.Has(page.Request.JenisKode)
	attachments, err := s.db.QueryContext(ctx, `SELECT id::text, jenis, mime, ukuran, dibuat_pada FROM lampiran WHERE permohonan_id=$1 ORDER BY dibuat_pada`, page.Request.ID)
	if err != nil {
		http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	for attachments.Next() {
		var attachment domain.Lampiran
		if err := attachments.Scan(&attachment.ID, &attachment.Jenis, &attachment.MIME, &attachment.Ukuran, &attachment.DibuatPada); err != nil {
			attachments.Close()
			http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		page.Request.Lampiran = append(page.Request.Lampiran, attachment)
	}
	if err := attachments.Err(); err != nil {
		attachments.Close()
		http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	attachments.Close()
	logs, err := s.db.QueryContext(ctx, `SELECT sl.status_lama, sl.status_baru, COALESCE(u.username, sl.aktor), sl.waktu, sl.catatan FROM status_log sl LEFT JOIN admin_user u ON u.id=sl.admin_id WHERE sl.permohonan_id=$1 ORDER BY sl.waktu DESC`, page.Request.ID)
	if err != nil {
		http.Error(w, "Riwayat status belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	for logs.Next() {
		var log domain.StatusLog
		if err := logs.Scan(&log.StatusLama, &log.StatusBaru, &log.Admin, &log.Waktu, &log.Catatan); err != nil {
			logs.Close()
			http.Error(w, "Riwayat status belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		page.Request.Riwayat = append(page.Request.Riwayat, log)
	}
	if err := logs.Err(); err != nil {
		logs.Close()
		http.Error(w, "Riwayat status belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	logs.Close()
	s.render(w, "detail.html", page)
}

func (s *Server) attachment(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var key, mime string
	var size int64
	err := s.db.QueryRowContext(ctx, `SELECT storage_key, mime, ukuran FROM lampiran WHERE id=$1`, r.PathValue("id")).Scan(&key, &mime, &size)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Lampiran belum dapat dibuka.", http.StatusServiceUnavailable)
		return
	}
	file, err := s.files.Open(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", fmt.Sprint(size))
	w.Header().Set("Content-Disposition", "inline; filename=\"lampiran\"")
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = io.Copy(w, file)
}

func (s *Server) updateIdentity(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	var kode string
	if err := s.db.QueryRowContext(r.Context(),
		`SELECT j.kode FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1`,
		r.PathValue("id")).Scan(&kode); err != nil {
		http.NotFound(w, r)
		return
	}
	identity := domain.Identitas{Tambahan: map[string]string{}}
	for _, field := range formFields(kode, identity) {
		value := strings.TrimSpace(r.FormValue("field-" + field.Key))
		if utf8.RuneCountInString(value) > 300 {
			http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=identity-error", http.StatusSeeOther)
			return
		}
		setIdentityValue(&identity, field.Key, value)
	}
	if err := validateIdentity(identity); err != nil {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=identity-error", http.StatusSeeOther)
		return
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(ctx, `UPDATE permohonan SET data_identitas=$1::jsonb, diubah_pada=now() WHERE id=$2 AND status NOT IN ('Selesai','Ditolak')`, string(encoded), r.PathValue("id"))
	if err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=status-error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=identity-saved", http.StatusSeeOther)
}

func setIdentityValue(identity *domain.Identitas, key, value string) {
	switch key {
	case "Nama":
		identity.Nama = value
	case "JenisKelamin":
		identity.JenisKelamin = value
	case "TempatLahir":
		identity.TempatLahir = value
	case "TanggalLahir":
		identity.TanggalLahir = value
	case "Nik":
		identity.NIK = value
	case "Agama":
		identity.Agama = value
	case "Pekerjaan":
		identity.Pekerjaan = value
	case "Alamat":
		identity.Alamat = value
	default:
		identity.Tambahan[key] = value
	}
}

func (s *Server) updateStatus(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	newStatus := domain.Status(strings.TrimSpace(r.FormValue("status")))
	note := strings.TrimSpace(r.FormValue("catatan"))
	if !validStatus(string(newStatus)) || utf8.RuneCountInString(note) > 500 {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=status-error", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()
	var oldStatus, kode string
	var identityRaw []byte
	err = tx.QueryRowContext(ctx, `SELECT p.status, p.data_identitas, j.kode FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1 FOR UPDATE OF p`, r.PathValue("id")).Scan(&oldStatus, &identityRaw, &kode)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
		return
	}
	if !canTransition(domain.Status(oldStatus), newStatus) {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=status-error", http.StatusSeeOther)
		return
	}
	if newStatus == domain.Dikerjakan {
		var identity domain.Identitas
		if json.Unmarshal(identityRaw, &identity) != nil || !identityComplete(kode, identity) {
			http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=identity-required", http.StatusSeeOther)
			return
		}
	}
	if newStatus == domain.Selesai {
		var identity domain.Identitas
		if json.Unmarshal(identityRaw, &identity) != nil || !letterFieldsComplete(kode, identity) {
			http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=letter-fields-required", http.StatusSeeOther)
			return
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE permohonan SET status=$1, diubah_pada=now(), selesai_pada=CASE WHEN $1='Selesai' THEN now() ELSE selesai_pada END WHERE id=$2`, newStatus, r.PathValue("id")); err != nil {
		http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
		return
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO status_log(permohonan_id,status_lama,status_baru,admin_id,catatan) VALUES ($1,$2,$3,$4,$5)`, r.PathValue("id"), oldStatus, newStatus, currentAdmin(r).ID, note); err != nil {
		http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
		return
	}
	if newStatus == domain.Selesai {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifikasi_log(permohonan_id,status_terkirim) VALUES ($1,'Selesai') ON CONFLICT (permohonan_id,status_terkirim) DO NOTHING`, r.PathValue("id")); err != nil {
			http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "Status belum dapat diperbarui.", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/requests/"+r.PathValue("id")+"?notice=status-saved", http.StatusSeeOther)
}

func validStatus(value string) bool {
	for _, status := range domain.Statuses {
		if value == string(status) {
			return true
		}
	}
	return false
}

func canTransition(from, to domain.Status) bool {
	switch from {
	case domain.Menunggu:
		return to == domain.Dikerjakan || to == domain.PerluPerbaikan || to == domain.Ditolak
	case domain.PerluPerbaikan:
		return to == domain.Dikerjakan || to == domain.Ditolak
	case domain.Dikerjakan:
		return to == domain.Selesai || to == domain.PerluPerbaikan || to == domain.Ditolak
	default:
		return false
	}
}

func validateIdentity(identity domain.Identitas) error {
	for _, value := range []struct {
		name string
		text string
		max  int
	}{
		{"nama", identity.Nama, 120},
		{"tempat lahir", identity.TempatLahir, 80},
		{"tanggal lahir", identity.TanggalLahir, 10},
		{"jenis kelamin", identity.JenisKelamin, 20},
		{"agama", identity.Agama, 40},
		{"NIK", identity.NIK, 16},
		{"pekerjaan", identity.Pekerjaan, 80},
		{"alamat", identity.Alamat, 300},
	} {
		if utf8.RuneCountInString(value.text) > value.max {
			return fmt.Errorf("%s too long", value.name)
		}
	}
	if identity.NIK != "" && !nikPattern.MatchString(identity.NIK) {
		return errors.New("NIK must contain 16 digits")
	}
	for key, value := range identity.Tambahan {
		if strings.HasSuffix(key, "Nik") && value != "" && !nikPattern.MatchString(value) {
			return errors.New("invalid NIK")
		}
	}
	for _, date := range []string{identity.TanggalLahir, identity.Tambahan["TanggalMeninggal"]} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return errors.New("invalid date")
			}
		}
	}
	if identity.JenisKelamin != "" && identity.JenisKelamin != "Laki-laki" && identity.JenisKelamin != "Perempuan" {
		return errors.New("invalid gender")
	}
	return nil
}

func identityComplete(code string, identity domain.Identitas) bool {
	fields := letterFields[code]
	if len(fields) == 0 {
		fields = formFields(code, identity)
	}
	for _, field := range fields {
		switch field.Key {
		case "Nama", "JenisKelamin", "TempatLahir", "TanggalLahir", "Nik", "Agama", "Pekerjaan", "Alamat":
			if strings.TrimSpace(identityValue(identity, field.Key)) == "" {
				return false
			}
		}
	}
	return len(fields) > 0
}

func letterFieldsComplete(code string, identity domain.Identitas) bool {
	fields := letterFields[code]

	if len(fields) == 0 || validateIdentity(identity) != nil {
		return false
	}

	for _, field := range fields {
		// NoUrut (Nomor Surat) bersifat opsional
		if field.Key == "NoUrut" {
			continue
		}

		// Field lainnya tetap wajib diisi
		if strings.TrimSpace(identityValue(identity, field.Key)) == "" {
			return false
		}
	}

	return true
}

