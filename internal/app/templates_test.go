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

func TestSettingsFAQDeleteForm(t *testing.T) {
	s := testServer(t)
	page := settingsPage{
		CSRF: "token-uji",
		FAQs: []faqSetting{{ID: 7, Question: "Jam buka?", Answer: "08.00", Keywords: []string{"jam buka"}, Active: true}},
	}
	rec := httptest.NewRecorder()
	s.render(rec, "settings.html", page)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`action="/settings/faq/delete"`,
		`data-delete-faq="faq-delete-7"`,
		`id="faq-delete-7"`,
		`name="csrf" value="token-uji"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("halaman settings tidak memuat %q", want)
		}
	}
	if strings.Contains(body, "data-deactivate-faq") {
		t.Error("tombol lama (nonaktifkan lewat ikon hapus) masih ada")
	}
}

func TestAdminPagesLoadTypeScaleLast(t *testing.T) {
	s := testServer(t)
	pages := map[string]any{
		"dashboard.html": dashboardPage{Pages: 1, Page: 1},
		"settings.html":  settingsPage{},
	}
	for name, data := range pages {
		rec := httptest.NewRecorder()
		s.render(rec, name, data)
		body := rec.Body.String()
		scale := strings.Index(body, "/static/type-scale.css")
		figma := strings.Index(body, "/static/figma-ui.css")
		if scale < 0 || scale < figma {
			t.Errorf("%s: type-scale.css harus dimuat setelah figma-ui.css (scale=%d figma=%d)", name, scale, figma)
		}
	}
}

func TestTypeScaleFontPairing(t *testing.T) {
	raw, err := assets.ReadFile("static/type-scale.css")
	if err != nil {
		t.Fatalf("type-scale.css tidak tertanam: %v", err)
	}
	css := string(raw)
	for _, want := range []string{
		`--ts-font-head: "Plus Jakarta Sans"`,
		`--ts-font-body: Inter`,
		`.admin-page { font-family: var(--ts-font-body)`,
		`.admin-page .brand strong,`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("type-scale.css tidak memuat %q", want)
		}
	}
	// Seluruh font yang dirujuk harus punya berkas lokal (CSP hanya mengizinkan 'self').
	app, err := assets.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{`font-family:Inter;`, `font-family:"Plus Jakarta Sans";`} {
		if !strings.Contains(string(app), "@font-face{"+family) {
			t.Errorf("app.css tidak mendeklarasikan @font-face untuk %s", family)
		}
	}
	for _, file := range []string{"static/fonts/inter-latin-var.woff2", "static/fonts/plus-jakarta-sans-latin-var.woff2"} {
		if _, err := assets.ReadFile(file); err != nil {
			t.Errorf("berkas font %s tidak tertanam: %v", file, err)
		}
	}
}

func TestLoginPageLoadsTypeScale(t *testing.T) {
	s := testServer(t)
	rec := httptest.NewRecorder()
	s.render(rec, "login.html", map[string]string{"Error": ""})
	if !strings.Contains(rec.Body.String(), "/static/type-scale.css") {
		t.Error("halaman login belum memuat type-scale.css")
	}
}
