package app

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func staticTestHandler(t *testing.T) http.Handler {
	t.Helper()
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		t.Fatal(err)
	}
	return staticHandler(sub)
}

func TestStaticHandlerCacheHeaders(t *testing.T) {
	h := staticTestHandler(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("app.css status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control css = %q", got)
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css") {
		t.Errorf("Content-Type css = %q", rec.Header().Get("Content-Type"))
	}
	lastModified := rec.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Fatal("Last-Modified kosong; browser tidak bisa meminta ulang secara bersyarat")
	}

	// Permintaan bersyarat harus dijawab 304 tanpa isi.
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req.Header.Set("If-Modified-Since", lastModified)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Errorf("permintaan bersyarat: status = %d, body = %d byte; want 304 dan kosong", rec.Code, rec.Body.Len())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/fonts/inter-latin-var.woff2", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("font status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "font/woff2" {
		t.Errorf("Content-Type font = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=604800" {
		t.Errorf("Cache-Control font = %q", got)
	}
}

func TestStaticHandlerRejectsListingsAndBadPaths(t *testing.T) {
	h := staticTestHandler(t)
	for _, path := range []string{
		"/static/", "/static/fonts/", "/static/fonts", "/static/tidak-ada.css",
		"/static/../templates/login.html", "/static/fonts/../../go.mod", "/static//app.css",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
}
