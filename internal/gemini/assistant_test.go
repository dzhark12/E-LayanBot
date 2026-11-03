package gemini

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTest(t *testing.T, handler http.HandlerFunc) *Assistant {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("k", "m")
	c.baseURL = srv.URL + "/"
	return NewAssistant(c, "Contoh", 2, log.New(io.Discard, "", 0))
}

var facts = []Fact{{Question: "Jam buka?", Answer: "Senin-Jumat 08.00-15.00"}}

func ok(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"` + text + `"}]},"finishReason":"STOP"}]}`))
	}
}

func TestAnswerUsesModelText(t *testing.T) {
	a := newTest(t, ok("Buka Senin-Jumat 08.00-15.00."))
	if got := a.Answer(context.Background(), "u", "jam buka?", facts); !strings.Contains(got, "08.00") {
		t.Fatalf("got %q", got)
	}
}

func TestUnknownMarkerBecomesFixedMessage(t *testing.T) {
	a := newTest(t, ok("[TIDAK_ADA_INFO]"))
	if got := a.Answer(context.Background(), "u", "siapa presiden?", facts); got != MsgUnknown {
		t.Fatalf("got %q", got)
	}
}

func TestNIKNeverReachesGemini(t *testing.T) {
	called := false
	a := newTest(t, func(http.ResponseWriter, *http.Request) { called = true })
	if got := a.Answer(context.Background(), "u", "NIK saya 1234567890123456", facts); got != MsgNoPII || called {
		t.Fatalf("got %q called=%v", got, called)
	}
}

func TestNoFactsNoCall(t *testing.T) {
	called := false
	a := newTest(t, func(http.ResponseWriter, *http.Request) { called = true })
	if got := a.Answer(context.Background(), "u", "halo", nil); got != MsgUnknown || called {
		t.Fatalf("got %q called=%v", got, called)
	}
}

func TestPerUserRateLimit(t *testing.T) {
	a := newTest(t, ok("ok"))
	for i := 0; i < 2; i++ {
		a.Answer(context.Background(), "u", "q", facts)
	}
	if got := a.Answer(context.Background(), "u", "q", facts); got != MsgLimit {
		t.Fatalf("got %q", got)
	}
	if got := a.Answer(context.Background(), "other", "q", facts); got == MsgLimit {
		t.Fatal("limit must be per user")
	}
}

func TestQuotaAndServerErrorsAreFriendly(t *testing.T) {
	a := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"x"}}`))
	})
	if got := a.Answer(context.Background(), "u", "q", facts); got != MsgBusy {
		t.Fatalf("got %q", got)
	}
}

func TestKeySentInHeaderNotURL(t *testing.T) {
	a := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "k" || r.URL.RawQuery != "" {
			t.Errorf("key must be in header only: %v %q", r.Header, r.URL.RawQuery)
		}
		ok("ok")(w, r)
	})
	a.Answer(context.Background(), "u", "q", facts)
}

func TestLongReplyIsTruncated(t *testing.T) {
	a := newTest(t, ok(strings.Repeat("a", 2000)))
	if got := a.Answer(context.Background(), "u", "q", facts); len([]rune(got)) > maxAnswerRunes+2 {
		t.Fatalf("len %d", len([]rune(got)))
	}
}
