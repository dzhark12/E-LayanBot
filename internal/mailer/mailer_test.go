package mailer

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildPasswordResetMessage(t *testing.T) {
	m := New("smtp.example.com", 587, "user", "pass", "E-Layan <admin@example.com>", "http://localhost:8080")
	msgBytes, err := m.BuildPasswordResetMessage("operator@sabahbalau.desa.id", "operator1", "http://localhost:8080/reset-password?token=abcdef123456")
	if err != nil {
		t.Fatalf("BuildPasswordResetMessage returned error: %v", err)
	}

	msg := string(msgBytes)
	for _, want := range []string{
		"To: <operator@sabahbalau.desa.id>",
		"Subject: Pemulihan Kata Sandi Akun E-LAYAN Desa Sabah Balau",
		"multipart/alternative",
		"http://localhost:8080/reset-password?token=abcdef123456",
		"15 menit",
		"operator1",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected email to contain %q, but not found", want)
		}
	}
}

func TestBuildPasswordResetMessageInvalidEmail(t *testing.T) {
	m := New("", 587, "", "", "", "http://localhost:8080")
	_, err := m.BuildPasswordResetMessage("not-an-email", "user", "http://localhost:8080/reset-password?token=xyz")
	if err == nil {
		t.Error("expected error for invalid email address, got nil")
	}
}

func TestSendPasswordResetDevSimulation(t *testing.T) {
	// Jika host kosong, tidak boleh error dan simulasi log berjalan
	m := New("", 587, "", "", "", "http://localhost:8080")
	err := m.SendPasswordReset("test@example.com", "admin", "http://localhost:8080/reset-password?token=test")
	if err != nil {
		t.Fatalf("expected nil error on dev simulation mode, got: %v", err)
	}
}

