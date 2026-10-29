package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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

func letterData(i domain.Identitas, now time.Time) map[string]string {
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
	out, err := pdf.Render(ctx, kode, letterData(ident, time.Now()))
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
