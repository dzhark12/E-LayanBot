package service

import (
	"strings"
	"testing"

	"elayan.local/desa/internal/domain"
)

func sample(status domain.Status, note string) notification {
	return notification{
		Code:   "DM00001",
		Status: status,
		Name:   "Budi Santoso",
		Letter: "Surat Keterangan Domisili",
		Note:   note,
	}
}

func TestBuildStatusMessageMenunggu(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.Menunggu, "Berkas Anda dalam antrean verifikasi."))
	want := "Halo, *Budi Santoso*.\n\n" +
		"Pengajuan surat Anda telah diterima oleh Pemerintah Desa Sabah Balau.\n\n" +
		"Informasi Pengajuan:\n" +
		"- Jenis Surat: Surat Keterangan Domisili\n" +
		"- Nomor Pengajuan: DM00001\n" +
		"- Status: *MENUNGGU*\n\n" +
		"Catatan Admin:\n" +
		"Berkas Anda dalam antrean verifikasi.\n\n" +
		"Terima kasih telah menggunakan layanan E-Layan Desa Sabah Balau."
	if !ok || got != want {
		t.Fatalf("ok=%v\n got: %q\nwant: %q", ok, got, want)
	}
}

func TestBuildStatusMessageMenungguCatatanKosong(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.Menunggu, ""))
	if !ok || !strings.Contains(got, "Tidak ada catatan tambahan dari admin.") {
		t.Fatalf("fallback catatan gagal: %q", got)
	}
}

func TestBuildStatusMessageDikerjakan(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.Dikerjakan, "Estimasi proses pengerjaan 1 hari kerja."))
	want := "Halo, *Budi Santoso*.\n\n" +
		"Pengajuan surat Anda saat ini sedang diproses oleh perangkat Desa Sabah Balau.\n\n" +
		"Informasi Pengajuan:\n" +
		"- Jenis Surat: Surat Keterangan Domisili\n" +
		"- Nomor Pengajuan: DM00001\n" +
		"- Status: *DIKERJAKAN*\n\n" +
		"Catatan Admin:\n" +
		"Estimasi proses pengerjaan 1 hari kerja.\n\n" +
		"Mohon menunggu informasi selanjutnya. Terima kasih."
	if !ok || got != want {
		t.Fatalf("ok=%v\n got: %q\nwant: %q", ok, got, want)
	}
}

func TestBuildStatusMessageSelesai(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.Selesai, "Surat dapat diambil di kantor desa membawa KTP asli."))
	want := "Halo, *Budi Santoso*.\n\n" +
		"Kabar baik! Pengajuan surat Anda telah selesai diproses.\n\n" +
		"Informasi Pengajuan:\n" +
		"- Jenis Surat: Surat Keterangan Domisili\n" +
		"- Nomor Pengajuan: DM00001\n" +
		"- Status: *SELESAI*\n\n" +
		"Catatan Admin:\n" +
		"Surat dapat diambil di kantor desa membawa KTP asli.\n\n" +
		"Silakan mengambil surat di balai desa.\n\n" +
		"Bagaimana penilaian Anda terhadap layanan kami? Balas dengan angka *1* sampai *5* (1 = sangat kurang, 5 = sangat baik).\n\n" +
		"Terima kasih telah menggunakan layanan E-Layan Desa Sabah Balau."
	if !ok || got != want {
		t.Fatalf("ok=%v\n got: %q\nwant: %q", ok, got, want)
	}
}

func TestBuildStatusMessageDitolak(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.Ditolak, "Pemohon tidak terdaftar pada data kependudukan desa."))
	want := "Halo, *Budi Santoso*.\n\n" +
		"Mohon maaf, pengajuan surat Anda telah ditolak oleh admin.\n\n" +
		"Informasi Pengajuan:\n" +
		"- Jenis Surat: Surat Keterangan Domisili\n" +
		"- Nomor Pengajuan: DM00001\n" +
		"- Status: *DITOLAK*\n\n" +
		"Keterangan Admin:\n" +
		"Pemohon tidak terdaftar pada data kependudukan desa.\n\n" +
		"Silakan menghubungi perangkat desa apabila membutuhkan informasi lebih lanjut."
	if !ok || got != want {
		t.Fatalf("ok=%v\n got: %q\nwant: %q", ok, got, want)
	}
}

func TestBuildStatusMessagePerluPerbaikan(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.PerluPerbaikan, "Foto kartu keluarga buram, mohon upload ulang."))
	for _, want := range []string{
		"- Status: *PERLU PERBAIKAN*",
		"Foto kartu keluarga buram, mohon upload ulang.",
		"Silakan kirim foto yang diminta dengan caption KTP, KK, RT, atau LAINNYA.",
		"Jika sudah, ketik *SELESAI*.",
		"Berkas sebelumnya tetap tersimpan.",
		"ketik *PERBAIKI DM00001* untuk memilih pengajuan ini.",
		"Terima kasih.",
	} {
		if !ok || !strings.Contains(got, want) {
			t.Errorf("pesan perbaikan tidak memuat %q: %q", want, got)
		}
	}
	if _, ok := buildStatusMessage(sample(domain.Diperbaiki, "")); ok {
		t.Error("Diperbaiki tidak seharusnya mengirim instruksi perbaikan lagi")
	}
}

func TestNamaKosongMemakaiFallback(t *testing.T) {
	n := sample(domain.Menunggu, "Ok")
	n.Name = ""
	got, ok := buildStatusMessage(n)
	if !ok || !strings.HasPrefix(got, "Halo, *Warga*.") {
		t.Fatalf("fallback nama salah: %q", got)
	}
}

func TestJenisSuratKosongMemakaiFallback(t *testing.T) {
	n := sample(domain.Menunggu, "Ok")
	n.Letter = ""
	got, ok := buildStatusMessage(n)
	if !ok || !strings.Contains(got, "- Jenis Surat: -\n") {
		t.Fatalf("fallback jenis surat salah: %q", got)
	}
}

func TestCatatanMultibaris(t *testing.T) {
	got, ok := buildStatusMessage(sample(domain.PerluPerbaikan, "Poin perbaikan:\r\n1. Foto KTP jelas\r\n2. Pengantar RT"))
	if !ok || !strings.Contains(got, "Poin perbaikan:\n1. Foto KTP jelas\n2. Pengantar RT") {
		t.Fatalf("format baris catatan rusak: %q", got)
	}
}

func TestNotifiesDanNoteRequired(t *testing.T) {
	for _, s := range []domain.Status{
		domain.Menunggu,
		domain.Dikerjakan,
		domain.Selesai,
		domain.Ditolak,
		domain.PerluPerbaikan,
	} {
		if !Notifies(s) {
			t.Errorf("Notifies(%s) = false, want true", s)
		}
	}
	if Notifies(domain.Diperbaiki) {
		t.Error("Diperbaiki tidak seharusnya mengirim notifikasi perbaikan baru")
	}

	for status, want := range map[domain.Status]bool{
		domain.Menunggu:       false,
		domain.Dikerjakan:     false,
		domain.Selesai:        false,
		domain.Ditolak:        true,
		domain.Diperbaiki:     false,
		domain.PerluPerbaikan: true,
	} {
		if NoteRequired(status) != want {
			t.Errorf("NoteRequired(%s) = %v, want %v", status, NoteRequired(status), want)
		}
	}
}
