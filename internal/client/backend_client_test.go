package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
