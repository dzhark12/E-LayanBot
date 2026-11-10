package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elayan.local/desa/config"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(nil, config.Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestLoginPageLabelsMatchBackend(t *testing.T) {
	s := testServer(t)
	rec := httptest.NewRecorder()
	s.render(rec, "login.html", map[string]string{"Error": ""})
	body := rec.Body.String()
	if strings.Contains(body, "Username atau Email") || strings.Contains(body, "@sabahbalau") {
		t.Error("halaman login masih menyebut email, padahal login hanya memakai nama pengguna")
	}
	if !strings.Contains(body, "Nama pengguna") {
		t.Error("label nama pengguna tidak ditemukan")
	}
	if strings.Contains(body, "form-error-alert") {
		t.Error("kotak galat tampil padahal tidak ada galat")
	}
}

func TestLoginErrorTitle(t *testing.T) {
	s := testServer(t)

	rec := httptest.NewRecorder()
	s.render(rec, "login.html", map[string]string{"Error": "Nama pengguna atau kata sandi salah."})
	if !strings.Contains(rec.Body.String(), "Kredensial Tidak Sesuai") {
		t.Error("judul bawaan galat login hilang")
	}

	rec = httptest.NewRecorder()
	s.renderStatus(rec, http.StatusTooManyRequests, "login.html", map[string]string{
		"ErrorTitle": "Terlalu Banyak Percobaan", "Error": "Coba lagi sekitar 15 menit.",
	})
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Terlalu Banyak Percobaan") || strings.Contains(body, "Kredensial Tidak Sesuai") {
		t.Error("judul galat 429 tidak sesuai")
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestDashboardTabCountsIgnoreFilter(t *testing.T) {
	s := testServer(t)
	page := dashboardPage{
		Status: "Menunggu", Query: "domisili", Page: 1, Pages: 1,
		Total:    2, // hasil filter
		Menunggu: 2, Dikerjakan: 3, Selesai: 4, PerluPerbaikan: 5, Ditolak: 6,
		RequestCount: 20, // seluruh permohonan
	}
	rec := httptest.NewRecorder()
	s.render(rec, "dashboard.html", page)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`>Semua <span class="tab-count">20</span>`,
		`>Perlu perbaikan <span class="tab-count">5</span>`,
		`>Ditolak <span class="tab-count">6</span>`,
		`placeholder="Cari kode, jenis surat, atau ringkasan..."`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard tidak memuat %q", want)
		}
	}
	if strings.Contains(body, "Cari NIK") {
		t.Error("placeholder masih menjanjikan pencarian NIK")
	}
}
