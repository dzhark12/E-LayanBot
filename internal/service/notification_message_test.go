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
		"Silakan mengikuti petunjuk pengambilan atau pengunduhan surat yang tersedia pada sistem.\n\n" +
		"Terima kasih telah menggunakan layanan E-Layan Desa Sabah Balau."
	if !ok || got != want {
		t.Fatalf("ok=%v\n got: %q\nwant: %q", ok, got, want)
	}
}

func TestBuildStatusMessageDibatalkan(t *testing.T) {
	for _, status := range []domain.Status{domain.Dibatalkan, domain.Ditolak} {
		got, ok := buildStatusMessage(sample(status, "Pemohon tidak terdaftar pada data kependudukan desa."))
		want := "Halo, *Budi Santoso*.\n\n" +
			"Mohon maaf, pengajuan surat Anda telah dibatalkan oleh admin.\n\n" +
			"Informasi Pengajuan:\n" +
			"- Jenis Surat: Surat Keterangan Domisili\n" +
			"- Nomor Pengajuan: DM00001\n" +
			"- Status: *DIBATALKAN*\n\n" +
			"Keterangan Admin:\n" +
			"Pemohon tidak terdaftar pada data kependudukan desa.\n\n" +
			"Silakan menghubungi perangkat desa apabila membutuhkan informasi lebih lanjut."
		if !ok || got != want {
			t.Fatalf("status %s ok=%v\n got: %q\nwant: %q", status, ok, got, want)
		}
	}
}

func TestBuildStatusMessageDiperbaiki(t *testing.T) {
	for _, status := range []domain.Status{domain.Diperbaiki, domain.PerluPerbaikan} {
		got, ok := buildStatusMessage(sample(status, "Foto kartu keluarga buram, mohon upload ulang."))
		want := "Halo, *Budi Santoso*.\n\n" +
			"Pengajuan surat Anda memerlukan perbaikan sebelum dapat diproses lebih lanjut.\n\n" +
			"Informasi Pengajuan:\n" +
			"- Jenis Surat: Surat Keterangan Domisili\n" +
			"- Nomor Pengajuan: DM00001\n" +
			"- Status: *DIPERBAIKI*\n\n" +
			"Catatan Perbaikan dari Admin:\n" +
			"Foto kartu keluarga buram, mohon upload ulang.\n\n" +
			"Mohon melakukan perbaikan sesuai catatan yang diberikan, kemudian ajukan kembali atau lengkapi persyaratan melalui sistem.\n\n" +
			"Terima kasih atas kerja samanya."
		if !ok || got != want {
			t.Fatalf("status %s ok=%v\n got: %q\nwant: %q", status, ok, got, want)
		}
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
	got, ok := buildStatusMessage(sample(domain.Diperbaiki, "Poin perbaikan:\r\n1. Foto KTP jelas\r\n2. Pengantar RT"))
	if !ok || !strings.Contains(got, "Poin perbaikan:\n1. Foto KTP jelas\n2. Pengantar RT") {
		t.Fatalf("format baris catatan rusak: %q", got)
	}
}

func TestNotifiesDanNoteRequired(t *testing.T) {
	for _, s := range []domain.Status{
		domain.Menunggu,
		domain.Dikerjakan,
		domain.Selesai,
		domain.Dibatalkan,
		domain.Ditolak,
		domain.Diperbaiki,
		domain.PerluPerbaikan,
	} {
		if !Notifies(s) {
			t.Errorf("Notifies(%s) = false, want true", s)
		}
	}

	for status, want := range map[domain.Status]bool{
		domain.Menunggu:       false,
		domain.Dikerjakan:     false,
		domain.Selesai:        false,
		domain.Dibatalkan:     true,
		domain.Ditolak:        true,
		domain.Diperbaiki:     true,
		domain.PerluPerbaikan: true,
	} {
		if NoteRequired(status) != want {
			t.Errorf("NoteRequired(%s) = %v, want %v", status, NoteRequired(status), want)
		}
	}
}
