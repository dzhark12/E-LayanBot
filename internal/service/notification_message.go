package service

import (
	"fmt"
	"strings"

	"elayan.local/desa/internal/domain"
)

// notification adalah satu baris antrean notifikasi beserta data yang
// dibutuhkan untuk menyusun pesan WhatsApp.
type notification struct {
	ID     int64
	JID    string
	Code   string
	Status domain.Status
	Name   string
	Letter string
	Note   string
}

// NotifiedStatuses adalah seluruh status yang memicu notifikasi WhatsApp ke pemohon.
var NotifiedStatuses = []domain.Status{
	domain.Menunggu,
	domain.Dikerjakan,
	domain.Selesai,
	domain.Dibatalkan,
	domain.Ditolak,
	domain.Diperbaiki,
	domain.PerluPerbaikan,
}

// Notifies melaporkan apakah perubahan ke status tersebut mengirim notifikasi.
func Notifies(status domain.Status) bool {
	for _, item := range NotifiedStatuses {
		if item == status {
			return true
		}
	}
	return false
}

// NoteRequired melaporkan apakah admin wajib mengisi catatan untuk status ini
// (terutama Dibatalkan/Ditolak dan Diperbaiki/Perlu Perbaikan).
func NoteRequired(status domain.Status) bool {
	return isDibatalkan(status) || isDiperbaiki(status)
}

func isMenunggu(s domain.Status) bool {
	return s == domain.Menunggu
}

func isDikerjakan(s domain.Status) bool {
	return s == domain.Dikerjakan
}

func isSelesai(s domain.Status) bool {
	return s == domain.Selesai
}

func isDibatalkan(s domain.Status) bool {
	return s == domain.Dibatalkan || s == domain.Ditolak
}

func isDiperbaiki(s domain.Status) bool {
	return s == domain.Diperbaiki || s == domain.PerluPerbaikan
}

// waInline membersihkan teks satu baris agar tidak merusak format WhatsApp.
func waInline(value string) string {
	value = strings.NewReplacer("*", "", "~", "", "`", "", "_", " ", "\r", " ", "\n", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

// cleanNote menormalkan line ending catatan admin tanpa menghilangkan struktur paragraf.
func cleanNote(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

// buildStatusMessage menyusun teks WhatsApp resmi sesuai status pengajuan surat.
// Mengembalikan false bila status tidak dikenali.
func buildStatusMessage(n notification) (string, bool) {
	name := waInline(n.Name)
	if name == "" {
		name = "Warga"
	}
	letter := waInline(n.Letter)
	if letter == "" {
		letter = "-"
	}
	note := cleanNote(n.Note)
	if note == "" {
		if isDibatalkan(n.Status) {
			note = "Tidak ada keterangan tambahan dari admin."
		} else {
			note = "Tidak ada catatan tambahan dari admin."
		}
	}

	switch {
	case isMenunggu(n.Status):
		return fmt.Sprintf("Halo, *%s*.\n\n"+
			"Pengajuan surat Anda telah diterima oleh Pemerintah Desa Sabah Balau.\n\n"+
			"Informasi Pengajuan:\n"+
			"- Jenis Surat: %s\n"+
			"- Nomor Pengajuan: %s\n"+
			"- Status: *MENUNGGU*\n\n"+
			"Catatan Admin:\n%s\n\n"+
			"Terima kasih telah menggunakan layanan E-Layan Desa Sabah Balau.",
			name, letter, n.Code, note), true

	case isDikerjakan(n.Status):
		return fmt.Sprintf("Halo, *%s*.\n\n"+
			"Pengajuan surat Anda saat ini sedang diproses oleh perangkat Desa Sabah Balau.\n\n"+
			"Informasi Pengajuan:\n"+
			"- Jenis Surat: %s\n"+
			"- Nomor Pengajuan: %s\n"+
			"- Status: *DIKERJAKAN*\n\n"+
			"Catatan Admin:\n%s\n\n"+
			"Mohon menunggu informasi selanjutnya. Terima kasih.",
			name, letter, n.Code, note), true

	case isSelesai(n.Status):
		return fmt.Sprintf("Halo, *%s*.\n\n"+
			"Kabar baik! Pengajuan surat Anda telah selesai diproses.\n\n"+
			"Informasi Pengajuan:\n"+
			"- Jenis Surat: %s\n"+
			"- Nomor Pengajuan: %s\n"+
			"- Status: *SELESAI*\n\n"+
			"Catatan Admin:\n%s\n\n"+
			"Silakan mengikuti petunjuk pengambilan atau pengunduhan surat yang tersedia pada sistem.\n\n"+
			"Terima kasih telah menggunakan layanan E-Layan Desa Sabah Balau.",
			name, letter, n.Code, note), true

	case isDibatalkan(n.Status):
		return fmt.Sprintf("Halo, *%s*.\n\n"+
			"Mohon maaf, pengajuan surat Anda telah dibatalkan oleh admin.\n\n"+
			"Informasi Pengajuan:\n"+
			"- Jenis Surat: %s\n"+
			"- Nomor Pengajuan: %s\n"+
			"- Status: *DIBATALKAN*\n\n"+
			"Keterangan Admin:\n%s\n\n"+
			"Silakan menghubungi perangkat desa apabila membutuhkan informasi lebih lanjut.",
			name, letter, n.Code, note), true

	case isDiperbaiki(n.Status):
		return fmt.Sprintf("Halo, *%s*.\n\n"+
			"Pengajuan surat Anda memerlukan perbaikan sebelum dapat diproses lebih lanjut.\n\n"+
			"Informasi Pengajuan:\n"+
			"- Jenis Surat: %s\n"+
			"- Nomor Pengajuan: %s\n"+
			"- Status: *DIPERBAIKI*\n\n"+
			"Catatan Perbaikan dari Admin:\n%s\n\n"+
			"Mohon melakukan perbaikan sesuai catatan yang diberikan, kemudian ajukan kembali atau lengkapi persyaratan melalui sistem.\n\n"+
			"Terima kasih atas kerja samanya.",
			name, letter, n.Code, note), true
	}

	return "", false
}
