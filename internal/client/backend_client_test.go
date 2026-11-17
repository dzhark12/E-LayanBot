package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elayan.local/desa/internal/storage"
)

func TestNewBackendRejectsInvalidURLs(t *testing.T) {
	for _, raw := range []string{"", "ftp://example.test", "http://user:pass@example.test", "http://example.test/path?q=x"} {
		if _, err := NewBackend(raw, "token", nil); err == nil {
			t.Errorf("NewBackend(%q) accepted invalid URL", raw)
		}
	}
}

func TestLetterTypesUsesBearerTokenAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/letter-types" || r.Method != http.MethodGet {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `[{"code":"DM","name":"Surat Domisili","requirements":["KTP"]}]`)
	}))
	defer server.Close()

	backend, err := NewBackend(server.URL, "test-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	backend.http = server.Client()
	got, err := backend.LetterTypes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Code != "DM" || got[0].Requirements[0] != "KTP" {
		t.Fatalf("LetterTypes() = %+v", got)
	}
}

func TestCheckStatusMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	backend, err := NewBackend(server.URL, "test-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	backend.http = server.Client()
	if _, err := backend.CheckStatus(context.Background(), "DM00001", "6280000000000@s.whatsapp.net"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CheckStatus() error = %v, want ErrNotFound", err)
	}
}

func TestActiveRepairsUsesBearerTokenAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repairs/active" || r.Method != http.MethodPost {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `"waJid":"6280000000000@s.whatsapp.net"`) {
			http.Error(w, "missing phone", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `[{"code":"TM00009","note":"kurang surat RT"}]`)
	}))
	defer server.Close()

	backend, err := NewBackend(server.URL, "test-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	backend.http = server.Client()
	got, err := backend.ActiveRepairs(context.Background(), "6280000000000@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Code != "TM00009" || got[0].Note != "kurang surat RT" {
		t.Fatalf("ActiveRepairs() = %+v", got)
	}
}

func TestCompleteRepairUploadsFilesAndRequestID(t *testing.T) {
	const repairID = "00000000-0000-4000-8000-000000000001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repairs/TM00009/complete" || r.Method != http.MethodPost {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if r.FormValue("repair_request_id") != repairID || r.FormValue("wa_jid") != "6280000000000@s.whatsapp.net" || r.FormValue("attachment_types") != "RT" {
			http.Error(w, "missing repair data", http.StatusBadRequest)
			return
		}
		files := r.MultipartForm.File["attachments"]
		if len(files) != 1 {
			http.Error(w, "missing attachment", http.StatusBadRequest)
			return
		}
		file, err := files[0].Open()
		if err != nil {
			http.Error(w, "invalid attachment", http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil || string(data) != "test photo" {
			http.Error(w, "incorrect attachment", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const key = "00000000-0000-4000-8000-000000000002.jpg"
	if err := files.Create(key, []byte("test photo")); err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(server.URL, "test-token", files)
	if err != nil {
		t.Fatal(err)
	}
	backend.http = server.Client()
	err = backend.CompleteRepair(context.Background(), "TM00009", "6280000000000@s.whatsapp.net", repairID, []Attachment{
		{Key: key, Kind: "RT", MIME: "image/jpeg", Size: int64(len("test photo"))},
	})
	if err != nil {
		t.Fatalf("CompleteRepair() error = %v", err)
	}
}
