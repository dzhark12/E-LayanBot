package domain

import "time"

type Status string

const (
	Menunggu       Status = "Menunggu"
	Dikerjakan     Status = "Dikerjakan"
	PerluPerbaikan Status = "Perlu Perbaikan"
	Diperbaiki     Status = "Diperbaiki"
	Selesai        Status = "Selesai"
	Ditolak        Status = "Ditolak"
	Dibatalkan     Status = "Dibatalkan"
)

var Statuses = []Status{Menunggu, Dikerjakan, Diperbaiki, Selesai, Dibatalkan}

type Identitas struct {
	Nama         string `json:"nama"`
	TempatLahir  string `json:"tempatLahir"`
	TanggalLahir string `json:"tanggalLahir"`
	JenisKelamin string `json:"jenisKelamin"`
	Agama        string `json:"agama"`
	NIK          string `json:"nik"`
	Pekerjaan    string `json:"pekerjaan"`
	Alamat       string `json:"alamat"`
	Tambahan map[string]string `json:"tambahan,omitempty"`
}

type StatusLog struct {
	StatusLama string
	StatusBaru string
	Admin      string
	Waktu      time.Time
	Catatan    string
}

type Permohonan struct {
	ID         string
	Kode       string
	JenisKode  string
	WAJID      string
	JenisNama  string
	Ringkasan  string
	Status     Status
	Identitas  Identitas
	DibuatPada time.Time
	DiubahPada time.Time
	Lampiran   []Lampiran
	Riwayat    []StatusLog
}
