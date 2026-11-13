package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/domain"
	"elayan.local/desa/internal/pdf"
)

var wib = time.FixedZone("WIB", 7*3600)
var bulanID = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli",
	"Agustus", "September", "Oktober", "November", "Desember"}
var romawi = []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

func tanggalID(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + bulanID[t.Month()] + " " + strconv.Itoa(t.Year())
}

func defaultSigner(code string) string {
	if code == "KM" {
		return SignerKades
	}
	return SignerSekdes
}

// Nilai hanya berasal dari konstanta di sini, bukan input pengguna.
func signerBlock(signer, tanggal string) (jabatan, nama string) {
	if signer == SignerKades {
		return `Sabah Balau, ` + tanggal + `\par Kepala Desa Sabah Balau`, "PUJIANTO, SE.,MM."
	}
	return `Sabah Balau, ` + tanggal + `\par Sekretaris Desa Sabah Balau`, "ERPANDI, S.Ag"
}

func letterData(code string, i domain.Identitas, now time.Time) map[string]string {
	now = now.In(wib)
	lahir := i.TanggalLahir
	if t, err := time.Parse("2006-01-02", i.TanggalLahir); err == nil {
		lahir = tanggalID(t)
	}
	d := map[string]string{
		"Nama": i.Nama, "JenisKelamin": i.JenisKelamin, "Nik": i.NIK, "Agama": i.Agama,
		"Pekerjaan": i.Pekerjaan, "Alamat": i.Alamat,
		"TempatTanggalLahir": i.TempatLahir + ", " + lahir,
		"Tanggal":            tanggalID(now),
		"BulanRomawi":        romawi[now.Month()],
		"Tahun":              strconv.Itoa(now.Year()),
		"Kecamatan":          "Tanjung Bintang", // nilai bawaan; bisa ditimpa lewat kolom tambahan
	}
	for k, v := range i.Tambahan {
		d[k] = v
	}

	signer := d["Penandatangan"]
	if signer != SignerKades && signer != SignerSekdes {
		signer = defaultSigner(code)
	}
	d["TtdJabatan"], d["TtdNama"] = signerBlock(signer, d["Tanggal"])
	if t, err := time.Parse("2006-01-02", d["TanggalMeninggal"]); err == nil {
		d["TanggalMeninggal"] = tanggalID(t)
	}
	return d
}

func (s *Server) letterPDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	var kode, requestCode, status string
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT j.kode, p.kode_unik, p.status, p.data_identitas FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1`, id).Scan(&kode, &requestCode, &status, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	if status != string(domain.Selesai) {
		http.Error(w, "PDF hanya tersedia setelah status permohonan selesai.", http.StatusConflict)
		return
	}
	var ident domain.Identitas
	if json.Unmarshal(raw, &ident) != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	if !letterFieldsComplete(kode, ident) {
		http.Error(w, "Lengkapi data surat terlebih dahulu.", http.StatusUnprocessableEntity)
		return
	}
	out, err := pdf.Render(ctx, kode, letterData(kode, ident, time.Now()))
	switch {
	case errors.Is(err, pdf.ErrNoTemplate):
		http.Error(w, "Belum ada template surat untuk jenis ini.", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="surat-`+requestCode+`.pdf"`)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(out)
}

func (s *Server) regenerateLetterPDF(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	s.letterPDF(w, r)
}

// letterFormFields mengembalikan field-field untuk form edit data surat
func letterFormFields(code string, identity domain.Identitas) []letterField {
	fields := append([]letterField(nil), letterFields[code]...)
	if len(fields) == 0 {
		fields = []letterField{
			{"NoUrut", "Nomor surat", "text", ""}, {"Nama", "Nama lengkap", "text", ""},
			{"TempatLahir", "Tempat lahir", "text", ""}, {"TanggalLahir", "Tanggal lahir", "date", ""},
			{"JenisKelamin", "Jenis kelamin", "gender", ""}, {"Nik", "NIK", "nik", ""},
			{"Agama", "Agama", "text", ""}, {"Pekerjaan", "Pekerjaan", "text", ""},
			{"Alamat", "Alamat", "textarea", ""}, {"Kecamatan", "Kecamatan", "text", "Tanjung Bintang"},
		}
	}
	// Penandatangan sudah ada di letterFields[code] (tipe "signer"), jangan tambah lagi
	for i := range fields {
		if value := identityValue(identity, fields[i].Key); value != "" {
			fields[i].Value = value
		}
	}
	return fields
}

// regenerateLetterForm menampilkan form untuk mengisi ulang data surat
func (s *Server) regenerateLetterForm(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var kode, requestCode, status string
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT j.kode, p.kode_unik, p.status, p.data_identitas FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1`, r.PathValue("id")).Scan(&kode, &requestCode, &status, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	if status != string(domain.Selesai) {
		http.Error(w, "PDF hanya tersedia setelah status permohonan selesai.", http.StatusConflict)
		return
	}
	var ident domain.Identitas
	if json.Unmarshal(raw, &ident) != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}

	// Ambil lampiran
	attachments, err := s.db.QueryContext(ctx, `SELECT id::text, jenis, mime, ukuran, dibuat_pada FROM lampiran WHERE permohonan_id=$1 ORDER BY dibuat_pada`, r.PathValue("id"))
	if err != nil {
		http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	var lampiran []domain.Lampiran
	for attachments.Next() {
		var att domain.Lampiran
		if err := attachments.Scan(&att.ID, &att.Jenis, &att.MIME, &att.Ukuran, &att.DibuatPada); err != nil {
			attachments.Close()
			http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		lampiran = append(lampiran, att)
	}
	if err := attachments.Err(); err != nil {
		attachments.Close()
		http.Error(w, "Lampiran belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	attachments.Close()

	// Render form untuk edit data surat
	user := currentAdmin(r)
	type regenPage struct {
		User     adminUser
		CSRF     string
		Request  domain.Permohonan
		Fields   []letterField
		HasPDF   bool
		Notice   string
		Lampiran []domain.Lampiran
	}
	page := regenPage{
		User:     user,
		CSRF:     user.CSRF,
		Fields:   letterFormFields(kode, ident),
		HasPDF:   pdf.Has(kode),
		Lampiran: lampiran,
		Request: domain.Permohonan{
			ID:        r.PathValue("id"),
			Kode:      requestCode,
			JenisKode: kode,
			Status:    domain.Status(status),
		},
	}
	s.render(w, "regenerate_letter.html", page)
}

// updateLetterFields menyimpan data surat yang diubah dan mencatat ke riwayat
func (s *Server) updateLetterFields(w http.ResponseWriter, r *http.Request) {
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Ambil kode jenis surat
	var kode string
	if err := s.db.QueryRowContext(ctx,
		`SELECT j.kode FROM permohonan p JOIN jenis_surat j ON j.id=p.jenis_id WHERE p.id=$1`,
		r.PathValue("id")).Scan(&kode); err != nil {
		http.NotFound(w, r)
		return
	}

	// Ambil identitas saat ini
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data_identitas FROM permohonan WHERE id=$1`, r.PathValue("id")).Scan(&raw); err != nil {
		http.Error(w, "Permohonan belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	var ident domain.Identitas
	if json.Unmarshal(raw, &ident) != nil {
		ident = domain.Identitas{Tambahan: map[string]string{}}
	}

	// Update field-field dari form
	fields := letterFormFields(kode, ident)
	for _, field := range fields {
		value := strings.TrimSpace(r.FormValue("field-" + field.Key))
		if utf8.RuneCountInString(value) > 300 {
			http.Redirect(w, r, "/requests/"+r.PathValue("id")+"/letter/regenerate?notice=identity-error", http.StatusSeeOther)
			return
		}
		setIdentityValue(&ident, field.Key, value)
	}

	if err := validateIdentity(ident); err != nil {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"/letter/regenerate?notice=identity-error", http.StatusSeeOther)
		return
	}

	if !letterFieldsComplete(kode, ident) {
		http.Redirect(w, r, "/requests/"+r.PathValue("id")+"/letter/regenerate?notice=letter-fields-required", http.StatusSeeOther)
		return
	}

	encoded, err := json.Marshal(ident)
	if err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusInternalServerError)
		return
	}

	// Simpan ke database
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE permohonan SET data_identitas=$1::jsonb, diubah_pada=now() WHERE id=$2`, string(encoded), r.PathValue("id")); err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}

	// Catat ke riwayat status
	if _, err := tx.ExecContext(ctx, `INSERT INTO status_log(permohonan_id,status_lama,status_baru,admin_id,catatan) VALUES ($1,'Selesai','Selesai',$2,'Mengenerate ulang data surat')`, r.PathValue("id"), currentAdmin(r).ID); err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Data identitas belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}

	// Redirect ke download PDF
	http.Redirect(w, r, "/requests/"+r.PathValue("id")+"/letter.pdf", http.StatusSeeOther)
}
