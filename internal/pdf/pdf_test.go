package pdf

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestHasKnownLetterTemplates(t *testing.T) {
	for _, code := range []string{"DM", "SU", "TM", "KM"} {
		if !Has(code) {
			t.Errorf("Has(%q) = false", code)
		}
	}
	if Has("ZZ") {
		t.Fatal("Has(ZZ) = true")
	}
}

func TestRenderRejectsUnknownTemplateAndMissingFields(t *testing.T) {
	if _, err := Render(context.Background(), "ZZ", nil); !errors.Is(err, ErrNoTemplate) {
		t.Fatalf("Render(unknown) error = %v, want ErrNoTemplate", err)
	}
	if _, err := Render(context.Background(), "DM", nil); err == nil || !strings.Contains(err.Error(), "data belum lengkap") {
		t.Fatalf("Render(incomplete) error = %v, want missing-fields error", err)
	}
}
