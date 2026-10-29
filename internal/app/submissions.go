package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"elayan.local/desa/internal/id"
	"elayan.local/desa/internal/pdf"
	"elayan.local/desa/pkg/validator"
)

var (
	letterCodePattern  = regexp.MustCompile(`^[A-Z]{2}$`)
	requestCodePattern = regexp.MustCompile(`^[A-Z]{2}[0-9]{5}$`)
	waJIDPattern       = regexp.MustCompile(`^[0-9]{5,20}(:[0-9]+)?@s\.whatsapp\.net$`)
)

type uploadedFile struct {
	id, kind, key, mime string
	data                []byte
}

func (s *Server) createSubmission(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "form permohonan tidak valid atau terlalu besar")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	letterCode := strings.ToUpper(strings.TrimSpace(r.FormValue("letter_code")))
	waJID := strings.TrimSpace(r.FormValue("wa_jid"))
	summary := strings.TrimSpace(r.FormValue("summary"))
	botRequestID := strings.ToLower(strings.TrimSpace(r.FormValue("request_id")))
	if !letterCodePattern.MatchString(letterCode) || !waJIDPattern.MatchString(waJID) || !uuidPattern.MatchString(botRequestID) || len(summary) < 1 || len(summary) > 500 {
		writeAPIError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "jenis surat, nomor WhatsApp, atau ringkasan tidak valid")
		return
	}
	waJID = normalizeWAJID(waJID)
	waNumberHash := s.requestHash([]byte("wa:" + strings.TrimSuffix(waJID, "@s.whatsapp.net")))
	if r.FormValue("consent_version") != "v1" {
		writeAPIError(w, http.StatusUnprocessableEntity, "CONSENT_REQUIRED", "persetujuan pemrosesan data wajib diberikan")
		return
	}
	var existingCode, existingStatus string
	var existingPhoneHash []byte
	err := s.db.QueryRowContext(r.Context(), `SELECT kode_unik,status,wa_number_hash FROM permohonan WHERE bot_request_id=$1`, botRequestID).Scan(&existingCode, &existingStatus, &existingPhoneHash)
	if err == nil {
		if subtle.ConstantTimeCompare(existingPhoneHash, waNumberHash) != 1 {
			writeAPIError(w, http.StatusConflict, "REQUEST_CONFLICT", "permohonan tidak dapat diproses")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"code": existingCode, "status": existingStatus})
		return
	}
	if err != sql.ErrNoRows {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	if !pdf.Has(letterCode) {
		writeAPIError(w, http.StatusUnprocessableEntity, "UNKNOWN_LETTER_TYPE", "jenis surat tidak tersedia")
		return
	}
	files := r.MultipartForm.File["attachments"]
	kinds := r.MultipartForm.Value["attachment_types"]
	if len(files) == 0 || len(files) > 5 || len(files) != len(kinds) {
		writeAPIError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "kirim 1 sampai 5 foto dan jenis lampiran yang sesuai")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var letterID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM jenis_surat WHERE kode=$1 AND aktif=TRUE`, letterCode).Scan(&letterID); err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusUnprocessableEntity, "UNKNOWN_LETTER_TYPE", "jenis surat tidak tersedia")
			return
		}
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}

	attachments := make([]uploadedFile, 0, len(files))
	cleanup := func() {
		for _, file := range attachments {
			if file.key != "" {
				_ = s.files.Remove(file.key)
			}
		}
	}
	for i, header := range files {
		kind := strings.ToUpper(strings.TrimSpace(kinds[i]))
		if kind != "KTP" && kind != "KK" && kind != "RT" && kind != "LAINNYA" {
			cleanup()
			writeAPIError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "jenis lampiran tidak valid")
			return
		}
		file, err := header.Open()
		if err != nil {
			cleanup()
			writeAPIError(w, http.StatusBadRequest, "INVALID_FILE", "foto tidak dapat dibaca")
			return
		}
		data, readErr := io.ReadAll(io.LimitReader(file, s.cfg.MaxAttachment+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) > s.cfg.MaxAttachment {
			cleanup()
			writeAPIError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "ukuran tiap foto maksimal 10 MB")
			return
		}
		mime, ext, err := imageType(data)
		if err != nil {
			cleanup()
			writeAPIError(w, http.StatusUnprocessableEntity, "INVALID_FILE", "lampiran harus berupa foto JPEG atau PNG yang valid")
			return
		}
		fileID, err := id.UUID()
		if err != nil {
			cleanup()
			writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
			return
		}
		storageID, err := id.UUID()
		if err != nil {
			cleanup()
			writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
			return
		}
		key := storageID + ext
		if err := s.files.Create(key, data); err != nil {
			cleanup()
			writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "lampiran belum dapat disimpan")
			return
		}
		attachments = append(attachments, uploadedFile{id: fileID, kind: kind, key: key, mime: mime, data: data})
	}
	cleanupFiles := true
	defer func() {
		if cleanupFiles {
			cleanup()
		}
	}()

	requestID, err := id.UUID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	defer tx.Rollback()
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('permohonan_code_seq')`).Scan(&sequence); err != nil || sequence > 99999 {
		writeAPIError(w, http.StatusServiceUnavailable, "CODE_UNAVAILABLE", "kode permohonan belum dapat dibuat")
		return
	}
	code := fmt.Sprintf("%s%05d", letterCode, sequence)
	if _, err := tx.ExecContext(ctx, `INSERT INTO permohonan(id, kode_unik, jenis_id, wa_jid, wa_number_hash, ringkasan, consent_version, consented_at, bot_request_id) VALUES ($1,$2,$3,$4,$5,$6,'v1',now(),$7)`, requestID, code, letterID, waJID, waNumberHash, summary, botRequestID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO status_log(permohonan_id,status_lama,status_baru,aktor) VALUES ($1,'-', 'Menunggu', 'Bot WhatsApp')`, requestID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	for _, file := range attachments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO lampiran(id, permohonan_id, jenis, storage_key, mime, ukuran) VALUES ($1,$2,$3,$4,$5,$6)`, file.id, requestID, file.kind, file.key, file.mime, len(file.data)); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		cleanupFiles = false
		writeAPIError(w, http.StatusInternalServerError, "UNAVAILABLE", "permohonan belum dapat disimpan")
		return
	}
	cleanupFiles = false
	writeJSON(w, http.StatusCreated, map[string]string{"code": code, "status": "Menunggu"})
}

func (s *Server) lookupStatus(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	var input struct {
		Code  string `json:"code"`
		WAJID string `json:"waJid"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "permintaan tidak valid")
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "permintaan tidak valid")
		return
	}
	input.Code = strings.ToUpper(strings.Join(strings.Fields(input.Code), ""))
	input.WAJID = strings.TrimSpace(input.WAJID)
	if !requestCodePattern.MatchString(input.Code) || !waJIDPattern.MatchString(input.WAJID) {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "kode tidak ditemukan")
		return
	}
	input.WAJID = normalizeWAJID(input.WAJID)
	phoneHash := s.requestHash([]byte("wa:" + strings.TrimSuffix(input.WAJID, "@s.whatsapp.net")))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	waHash := s.requestHash([]byte("lookup:" + strings.TrimSuffix(input.WAJID, "@s.whatsapp.net")))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockID(waHash)); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM status_lookup_attempt WHERE attempted_at < now() - interval '1 hour'`); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO status_lookup_attempt(wa_hash) VALUES ($1)`, waHash); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	var attempts int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM status_lookup_attempt WHERE wa_hash=$1 AND attempted_at > now() - interval '1 hour'`, waHash).Scan(&attempts); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	if err := tx.Commit(); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "status belum dapat diperiksa")
		return
	}
	if attempts > 10 {
		writeAPIError(w, http.StatusTooManyRequests, "RATE_LIMITED", "terlalu banyak percobaan")
		return
	}
	var code, status string
	var storedPhoneHash []byte
	err = s.db.QueryRowContext(ctx, `SELECT kode_unik, status, wa_number_hash FROM permohonan WHERE kode_unik=$1`, input.Code).Scan(&code, &status, &storedPhoneHash)
	if err != nil || subtle.ConstantTimeCompare(storedPhoneHash, phoneHash) != 1 {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "kode tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": code, "status": status})
}

func normalizeWAJID(jid string) string {
	local := strings.TrimSuffix(jid, "@s.whatsapp.net")
	if device, _, ok := strings.Cut(local, ":"); ok {
		local = device
	}
	return local + "@s.whatsapp.net"
}

func imageType(data []byte) (mime, extension string, err error) {
	return validator.ImageType(data, 10<<20)
}
