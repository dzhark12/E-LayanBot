package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/security"
)

const sessionCookie = "elayansession"
const sessionDuration = 12 * time.Hour

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, err := s.session(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := map[string]any{"Error": ""}
	if r.URL.Query().Get("reset") == "success" {
		data["Notice"] = "Kata sandi berhasil direset. Silakan masuk dengan kata sandi baru."
	}
	s.render(w, "login.html", data)
}

func (s *Server) forgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login.html", map[string]any{"Reset": true, "ResetEnabled": s.cfg.AdminRecoveryCode != ""})
}

func (s *Server) resetForgottenPassword(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	data := map[string]any{
		"Reset": true, "ResetEnabled": s.cfg.AdminRecoveryCode != "",
		"Error": "Sistem belum dapat memproses reset. Coba lagi beberapa saat.",
	}
	if err := r.ParseForm(); err != nil {
		data["Error"] = "Permintaan tidak dapat diproses. Periksa isian dan kode pemulihan."
		s.render(w, "login.html", data)
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	code := r.FormValue("recovery_code")
	password, confirmation := r.FormValue("new_password"), r.FormValue("confirm_password")
	validInput := username != "" && len(username) <= 80 && len(code) <= 256 && utf8.RuneCountInString(password) >= 12 && len(password) <= 256 && password == confirmation

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ipHash := s.requestHash([]byte("password-reset:" + remoteIP(r)))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockID(ipHash)); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_login_attempt WHERE attempted_at < now() - interval '1 day'`); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	var attempts int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM admin_login_attempt WHERE ip_hash=$1 AND attempted_at > now() - interval '15 minutes'`, ipHash).Scan(&attempts); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	if attempts >= 8 {
		data["Error"] = "Reset belum dapat diproses. Coba lagi sekitar 15 menit."
		s.renderStatus(w, http.StatusTooManyRequests, "login.html", data)
		return
	}
	validCode := s.cfg.AdminRecoveryCode != "" && len(code) == len(s.cfg.AdminRecoveryCode) && subtle.ConstantTimeCompare([]byte(code), []byte(s.cfg.AdminRecoveryCode)) == 1
	var adminID int64
	if validInput && validCode {
		err = tx.QueryRowContext(ctx, `SELECT id FROM admin_user WHERE username=$1 AND aktif=TRUE`, username).Scan(&adminID)
		if err != nil && err != sql.ErrNoRows {
			s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
			return
		}
	}
	if !validInput || !validCode || adminID == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO admin_login_attempt(ip_hash) VALUES ($1)`, ipHash); err != nil {
			s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
			return
		}
		if err := tx.Commit(); err != nil {
			s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
			return
		}
		data["Error"] = "Reset belum berhasil. Periksa nama pengguna, kode pemulihan, dan kata sandi baru."
		s.render(w, "login.html", data)
		return
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	result, err := tx.ExecContext(ctx, `UPDATE admin_user SET password_hash=$1 WHERE id=$2 AND aktif=TRUE`, passwordHash, adminID)
	if err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		data["Error"] = "Reset belum berhasil. Periksa nama pengguna, kode pemulihan, dan kata sandi baru."
		s.render(w, "login.html", data)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_session WHERE admin_id=$1`, adminID); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_login_attempt WHERE ip_hash=$1`, ipHash); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	if err := tx.Commit(); err != nil {
		s.renderStatus(w, http.StatusServiceUnavailable, "login.html", data)
		return
	}
	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	// Galat ditampilkan di halaman login (bukan teks polos) dengan kode status yang tepat.
	fail := func(status int) {
		s.renderStatus(w, status, "login.html", map[string]string{
			"ErrorTitle": "Layanan Sedang Bermasalah",
			"Error":      "Sistem belum dapat memproses permintaan. Coba lagi beberapa saat.",
		})
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		s.render(w, "login.html", map[string]string{"Error": "Isian tidak valid."})
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	remember := r.FormValue("remember") == "on"
	if username == "" || password == "" || len(username) > 80 || len(password) > 256 {
		s.render(w, "login.html", map[string]string{"Error": "Nama pengguna atau kata sandi salah."})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ipHash := s.requestHash([]byte("login:" + remoteIP(r) + "\x00" + username))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		fail(http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockID(ipHash)); err != nil {
		fail(http.StatusServiceUnavailable)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_login_attempt WHERE attempted_at < now() - interval '1 day'`); err != nil {
		fail(http.StatusServiceUnavailable)
		return
	}
	var attempts int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM admin_login_attempt WHERE ip_hash=$1 AND attempted_at > now() - interval '15 minutes'`, ipHash).Scan(&attempts); err != nil {
		fail(http.StatusServiceUnavailable)
		return
	}
	if attempts >= 8 {
		s.renderStatus(w, http.StatusTooManyRequests, "login.html", map[string]string{
			"ErrorTitle": "Terlalu Banyak Percobaan",
			"Error":      "Demi keamanan, percobaan masuk dibatasi sementara. Coba lagi sekitar 15 menit.",
		})
		return
	}

	var user adminUser
	var passwordHash string
	err = tx.QueryRowContext(ctx, `SELECT id, username, role, password_hash FROM admin_user WHERE username=$1 AND aktif=TRUE`, username).Scan(&user.ID, &user.Username, &user.Role, &passwordHash)
	if err != nil && err != sql.ErrNoRows {
		fail(http.StatusServiceUnavailable)
		return
	}
	if err == sql.ErrNoRows || !security.VerifyPassword(password, passwordHash) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO admin_login_attempt(ip_hash) VALUES ($1)`, ipHash); err != nil {
			fail(http.StatusServiceUnavailable)
			return
		}
		if err := tx.Commit(); err != nil {
			fail(http.StatusServiceUnavailable)
			return
		}
		s.render(w, "login.html", map[string]string{"Error": "Nama pengguna atau kata sandi salah."})
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_login_attempt WHERE ip_hash=$1`, ipHash); err != nil {
		fail(http.StatusServiceUnavailable)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		fail(http.StatusInternalServerError)
		return
	}
	csrf, err := randomToken(32)
	if err != nil {
		fail(http.StatusInternalServerError)
		return
	}
	duration := sessionDuration
	if remember {
		duration = 30 * 24 * time.Hour
	}
	expires := time.Now().UTC().Add(duration)
	if _, err := tx.ExecContext(ctx, `INSERT INTO admin_session(token_hash, csrf_token, admin_id, expires_at) VALUES ($1,$2,$3,$4)`, s.sessionHash(token), csrf, user.ID, expires); err != nil {
		fail(http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(duration.Seconds())})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	user := currentAdmin(r)
	if !s.validCSRF(r, user.CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_, _ = s.db.ExecContext(r.Context(), `DELETE FROM admin_session WHERE token_hash=$1`, s.sessionHash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.session(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), adminContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) session(r *http.Request) (adminUser, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return adminUser{}, sql.ErrNoRows
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	var user adminUser
	err = s.db.QueryRowContext(ctx, `SELECT u.id, u.username, u.role, s.csrf_token FROM admin_session s JOIN admin_user u ON u.id=s.admin_id WHERE s.token_hash=$1 AND s.expires_at > now() AND u.aktif=TRUE`, s.sessionHash(cookie.Value)).Scan(&user.ID, &user.Username, &user.Role, &user.CSRF)
	return user, err
}

func currentAdmin(r *http.Request) adminUser {
	user, _ := r.Context().Value(adminContextKey).(adminUser)
	return user
}

func (s *Server) validCSRF(r *http.Request, expected string) bool {
	got := r.FormValue("csrf")
	return sameOrigin(r) && expected != "" && len(got) == len(expected) && subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Referer()
	}
	parsed, err := url.Parse(origin)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host == r.Host
}

func (s *Server) sessionHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func advisoryLockID(hash []byte) int64 {
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func (s *Server) requestHash(value []byte) []byte {
	hash := hmac.New(sha256.New, []byte(s.cfg.WAHashKey))
	hash.Write(value)
	return hash.Sum(nil)
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) botAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		token := strings.TrimPrefix(r.Header.Get("Authorization"), prefix)
		if !strings.HasPrefix(r.Header.Get("Authorization"), prefix) || len(token) != len(s.cfg.BotAPIToken) || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.BotAPIToken)) != 1 {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
			return
		}
		next(w, r)
	}
}
