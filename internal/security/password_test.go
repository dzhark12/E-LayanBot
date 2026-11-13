package security

import "testing"

func TestHashPasswordAndVerify(t *testing.T) {
	password := "contoh-password-aman"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == password || !VerifyPassword(password, hash) {
		t.Fatal("hash tidak bisa memverifikasi kata sandi yang benar")
	}
	if VerifyPassword("kata-sandi-salah", hash) {
		t.Fatal("hash menerima kata sandi yang salah")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, encoded := range []string{"", "invalid", "pbkdf2-sha256$abc$%%$%%", "other$600000$a$b"} {
		if VerifyPassword("password", encoded) {
			t.Errorf("VerifyPassword accepted malformed hash %q", encoded)
		}
	}
}
