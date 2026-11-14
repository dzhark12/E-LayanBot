package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/security"
)

type forgotPasswordPageData struct {
	Error      string
	ErrorTitle string
	Success    string
	Email      string
}

type resetPasswordPageData struct {
	Token      string
	Username   string
	Error      string
	ErrorTitle string
	Expired    bool
	Invalid    bool
}

// forgotPasswordPage menampilkan halaman permintaan reset kata sandi.
func (s *Server) forgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	if _, err := s.session(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "forgot_password.html", forgotPasswordPageData{})
}

// forgotPassword memproses formulir permintaan link reset password via email.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		s.render(w, "forgot_password.html", forgotPasswordPageData{
			Error: "Isian tidak valid.",
		})
		return
	}

	rawEmail := strings.TrimSpace(r.FormValue("email"))
	if rawEmail == "" {
		s.render(w, "forgot_password.html", forgotPasswordPageData{
			Error: "Alamat email wajib diisi.",
		})
		return
	}

	addr, err := mail.ParseAddress(rawEmail)
	if err != nil || !strings.Contains(rawEmail, "@") || !strings.Contains(rawEmail, ".") {
		s.render(w, "forgot_password.html", forgotPasswordPageData{
			Error: "Format alamat email tidak valid.",
			Email: rawEmail,
		})
		return
	}
	email := strings.ToLower(addr.Address)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ipHash := s.requestHash([]byte("forgot-ip:" + remoteIP(r)))
	targetHash := s.requestHash([]byte("forgot-email:" + email))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "forgot_password.html", forgotPasswordPageData{
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
			Email:      rawEmail,
		})
		return
	}
	defer tx.Rollback()

	// Rate limiting: bersihkan riwayat percobaan lama (> 1 hari)
	_, _ = tx.ExecContext(ctx, `DELETE FROM password_reset_attempt WHERE attempted_at < now() - interval '1 day'`)

	var ipAttempts, emailAttempts int
	_ = tx.QueryRowContext(ctx, `SELECT count(*) FROM password_reset_attempt WHERE target_hash=$1 AND attempted_at > now() - interval '15 minutes'`, ipHash).Scan(&ipAttempts)
	_ = tx.QueryRowContext(ctx, `SELECT count(*) FROM password_reset_attempt WHERE target_hash=$1 AND attempted_at > now() - interval '15 minutes'`, targetHash).Scan(&emailAttempts)

	if ipAttempts >= 5 || emailAttempts >= 5 {
		s.renderStatus(w, http.StatusTooManyRequests, "forgot_password.html", forgotPasswordPageData{
			ErrorTitle: "Terlalu Banyak Permintaan",
			Error:      "Demi keamanan, permintaan reset kata sandi dibatasi sementara. Coba lagi sekitar 15 menit.",
			Email:      rawEmail,
		})
		return
	}

	// Catat percobaan ke password_reset_attempt
	_, _ = tx.ExecContext(ctx, `INSERT INTO password_reset_attempt(target_hash) VALUES ($1)`, ipHash)
	_, _ = tx.ExecContext(ctx, `INSERT INTO password_reset_attempt(target_hash) VALUES ($1)`, targetHash)

	// Cari akun berdasarkan email terdaftar
	var userID int64
	var username string
	err = tx.QueryRowContext(ctx, `SELECT id, username FROM admin_user WHERE LOWER(email)=$1 AND aktif=TRUE`, email).Scan(&userID, &username)

	const genericSuccess = "Jika alamat email terdaftar pada sistem, kami akan mengirimkan tautan untuk mengatur ulang kata sandi Anda."

	if errors.Is(err, sql.ErrNoRows) {
		// Anti-enumeration: respon tetap sama meskipun email tidak terdaftar
		if err := tx.Commit(); err != nil {
			s.renderStatus(w, http.StatusServiceUnavailable, "forgot_password.html", forgotPasswordPageData{
				Error: "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
			})
			return
		}
		s.render(w, "forgot_password.html", forgotPasswordPageData{
			Success: genericSuccess,
		})
		return
	}
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "forgot_password.html", forgotPasswordPageData{
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
		return
	}

	// Buat token reset acak kriptografis 32 bytes (256-bit entropy)
	rawToken, err := randomToken(32)
	if err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "forgot_password.html", forgotPasswordPageData{
			Error: "Gagal membuat token keamanan.",
		})
		return
	}

	tokenHash := sha256.Sum256([]byte(rawToken))
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	if _, err := tx.ExecContext(ctx, `INSERT INTO password_reset_token(admin_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash[:], expiresAt); err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "forgot_password.html", forgotPasswordPageData{
			Error: "Gagal menyimpan data pemulihan kata sandi.",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "forgot_password.html", forgotPasswordPageData{
			Error: "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
		return
	}

	// Kirim email pemulihan
	baseURL := s.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, rawToken)

	go func(toEmail, toUser, link string) {
		if err := s.mailer.SendPasswordReset(toEmail, toUser, link); err != nil {
			log.Printf("[MAILER ERROR] Gagal mengirim email reset password ke %s: %v", toEmail, err)
		}
	}(email, username, resetURL)

	s.render(w, "forgot_password.html", forgotPasswordPageData{
		Success: genericSuccess,
	})
}

// resetPasswordPage menampilkan halaman formulir kata sandi baru jika token valid.
func (s *Server) resetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Tidak Lengkap",
			Error:      "Tautan pengaturan ulang kata sandi tidak memiliki token valid.",
		})
		return
	}

	tokenHash := sha256.Sum256([]byte(token))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var tokenID, adminID int64
	var username string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `SELECT t.id, t.admin_id, u.username, t.expires_at, t.used_at FROM password_reset_token t JOIN admin_user u ON u.id=t.admin_id WHERE t.token_hash=$1 AND u.aktif=TRUE`, tokenHash[:]).Scan(&tokenID, &adminID, &username, &expiresAt, &usedAt)

	if errors.Is(err, sql.ErrNoRows) {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Tidak Valid",
			Error:      "Tautan pengaturan ulang kata sandi tidak ditemukan atau sudah tidak berlaku.",
		})
		return
	}
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
		return
	}

	if usedAt.Valid {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Telah Digunakan",
			Error:      "Tautan ini sudah pernah digunakan sebelumnya. Setiap tautan hanya berlaku satu kali demi keamanan.",
		})
		return
	}

	if time.Now().UTC().After(expiresAt) {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Expired:    true,
			ErrorTitle: "Tautan Telah Kedaluwarsa",
			Error:      "Tautan pengaturan ulang kata sandi telah melebihi batas waktu 15 menit. Silakan ajukan permohonan baru.",
		})
		return
	}

	s.render(w, "reset_password.html", resetPasswordPageData{
		Token:    token,
		Username: username,
	})
}

// resetPassword memvalidasi dan menyimpan kata sandi baru.
func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid: true,
			Error:   "Isian tidak valid.",
		})
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if token == "" {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Tidak Valid",
			Error:      "Token pemulihan tidak ditemukan.",
		})
		return
	}

	tokenHash := sha256.Sum256([]byte(token))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Rate limiting submit reset password
	ipHash := s.requestHash([]byte("reset-submit-ip:" + remoteIP(r)))
	var attempts int
	_ = s.db.QueryRowContext(ctx, `SELECT count(*) FROM password_reset_attempt WHERE target_hash=$1 AND attempted_at > now() - interval '15 minutes'`, ipHash).Scan(&attempts)
	if attempts >= 10 {
		s.renderStatus(w, http.StatusTooManyRequests, "reset_password.html", resetPasswordPageData{
			Token:      token,
			ErrorTitle: "Terlalu Banyak Percobaan",
			Error:      "Percobaan dibatasi sementara demi keamanan. Coba lagi sekitar 15 menit.",
		})
		return
	}

	// Ambil username untuk keperluan tampilan jika validasi password gagal
	var preUsername string
	_ = s.db.QueryRowContext(ctx, `SELECT u.username FROM password_reset_token t JOIN admin_user u ON u.id=t.admin_id WHERE t.token_hash=$1`, tokenHash[:]).Scan(&preUsername)

	// Validasi panjang password: minimal 12 karakter
	if utf8.RuneCountInString(password) < 12 {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO password_reset_attempt(target_hash) VALUES ($1)`, ipHash)
		s.render(w, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   preUsername,
			ErrorTitle: "Kata Sandi Kurang Kuat",
			Error:      "Kata sandi baru minimal harus terdiri dari 12 karakter.",
		})
		return
	}

	if password != passwordConfirm {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO password_reset_attempt(target_hash) VALUES ($1)`, ipHash)
		s.render(w, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   preUsername,
			ErrorTitle: "Kata Sandi Tidak Cocok",
			Error:      "Kata sandi baru dan konfirmasi kata sandi tidak sama. Silakan periksa kembali.",
		})
		return
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   preUsername,
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
		return
	}
	defer tx.Rollback()

	// Kunci baris token untuk menghindari race condition / double-spending
	var tokenID, adminID int64
	var username string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err = tx.QueryRowContext(ctx, `SELECT t.id, t.admin_id, u.username, t.expires_at, t.used_at FROM password_reset_token t JOIN admin_user u ON u.id=t.admin_id WHERE t.token_hash=$1 AND u.aktif=TRUE FOR UPDATE OF t`, tokenHash[:]).Scan(&tokenID, &adminID, &username, &expiresAt, &usedAt)

	if errors.Is(err, sql.ErrNoRows) {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Tidak Valid",
			Error:      "Tautan pengaturan ulang kata sandi tidak valid atau akun dinonaktifkan.",
		})
		return
	}
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   preUsername,
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
		return
	}

	if usedAt.Valid {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Invalid:    true,
			ErrorTitle: "Tautan Telah Digunakan",
			Error:      "Tautan ini sudah pernah digunakan sebelumnya.",
		})
		return
	}

	if time.Now().UTC().After(expiresAt) {
		s.render(w, "reset_password.html", resetPasswordPageData{
			Expired:    true,
			ErrorTitle: "Tautan Telah Kedaluwarsa",
			Error:      "Tautan pengaturan ulang kata sandi telah kedaluwarsa. Silakan ajukan permohonan baru.",
		})
		return
	}

	// Hash password baru menggunakan mekanisme hashing bawaan sistem (PBKDF2-SHA256)
	newHash, err := security.HashPassword(password)
	if err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   username,
			ErrorTitle: "Kesalahan Sistem",
			Error:      "Gagal mengenkripsi kata sandi baru.",
		})
		return
	}

	// 1. Update password di admin_user
	if _, err := tx.ExecContext(ctx, `UPDATE admin_user SET password_hash=$1 WHERE id=$2`, newHash, adminID); err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   username,
			ErrorTitle: "Kesalahan Sistem",
			Error:      "Gagal memperbarui kata sandi.",
		})
		return
	}

	// 2. Tandai token ini sebagai sudah digunakan
	if _, err := tx.ExecContext(ctx, `UPDATE password_reset_token SET used_at=now() WHERE id=$1`, tokenID); err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   username,
			ErrorTitle: "Kesalahan Sistem",
			Error:      "Gagal memperbarui status token.",
		})
		return
	}

	// 3. Batalkan token reset aktif lainnya milik user ini
	_, _ = tx.ExecContext(ctx, `UPDATE password_reset_token SET used_at=now() WHERE admin_id=$1 AND used_at IS NULL AND id <> $2`, adminID, tokenID)

	// 4. Cabut SELURUH sesi login lama milik akun ini demi keamanan
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_session WHERE admin_id=$1`, adminID); err != nil {
		s.renderStatus(w, http.StatusInternalServerError, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   username,
			ErrorTitle: "Kesalahan Sistem",
			Error:      "Gagal mencabut sesi lama pengguna.",
		})
		return
	}

	// 5. Catat ke audit log
	_, _ = tx.ExecContext(ctx, `INSERT INTO admin_audit_log(admin_id, action, target_user_id, details, ip_address) VALUES ($1, 'password_reset_success', $1, 'Pengguna berhasil mengatur ulang kata sandi secara mandiri', $2)`, adminID, remoteIP(r))

	if err := tx.Commit(); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "reset_password.html", resetPasswordPageData{
			Token:      token,
			Username:   username,
			ErrorTitle: "Layanan Sedang Bermasalah",
			Error:      "Gagal menyelesaikan transaksi pembaharuan kata sandi.",
		})
		return
	}

	// Berhasil: arahkan pengguna kembali ke halaman login dengan notifikasi sukses
	http.Redirect(w, r, "/login?notice=password-reset-success", http.StatusSeeOther)
}

