package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"elayan.local/desa/internal/security"
)

type letterSetting struct {
	Code         string
	Name         string
	Requirements []string
	Active       bool
}

type faqSetting struct {
	ID       int64
	Question string
	Keywords []string
	Answer   string
	Category string
	Active   bool
}

type operatorUser struct {
	ID        int64
	Username  string
	Email     string
	Role      string
	Active    bool
	CreatedAt time.Time
}

type settingsPage struct {
	User      adminUser
	CSRF      string
	Letters   []letterSetting
	FAQs      []faqSetting
	Operators []operatorUser
	Notice    string
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user := currentAdmin(r)
	page := settingsPage{User: user, CSRF: user.CSRF, Notice: r.URL.Query().Get("notice")}
	letters, err := s.db.QueryContext(ctx, `SELECT kode,nama,array_to_json(syarat)::text,aktif FROM jenis_surat ORDER BY kode`)
	if err != nil {
		http.Error(w, "Jenis surat belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	for letters.Next() {
		var item letterSetting
		var requirements []byte
		if err := letters.Scan(&item.Code, &item.Name, &requirements, &item.Active); err != nil || json.Unmarshal(requirements, &item.Requirements) != nil {
			letters.Close()
			http.Error(w, "Jenis surat belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		page.Letters = append(page.Letters, item)
	}
	if err := letters.Err(); err != nil {
		letters.Close()
		http.Error(w, "Jenis surat belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	letters.Close()
	faqs, err := s.db.QueryContext(ctx, `SELECT id,pertanyaan,array_to_json(kata_kunci)::text,jawaban,kategori,aktif FROM faq ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "FAQ belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}
	defer faqs.Close()
	for faqs.Next() {
		var item faqSetting
		var keywords []byte
		if err := faqs.Scan(&item.ID, &item.Question, &keywords, &item.Answer, &item.Category, &item.Active); err != nil || json.Unmarshal(keywords, &item.Keywords) != nil {
			http.Error(w, "FAQ belum dapat dimuat.", http.StatusServiceUnavailable)
			return
		}
		page.FAQs = append(page.FAQs, item)
	}
	if err := faqs.Err(); err != nil {
		http.Error(w, "FAQ belum dapat dimuat.", http.StatusServiceUnavailable)
		return
	}

	if user.Role == "admin" {
		rows, err := s.db.QueryContext(ctx, `SELECT id, username, COALESCE(email, ''), role, aktif, dibuat_pada FROM admin_user ORDER BY id ASC`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var op operatorUser
				if err := rows.Scan(&op.ID, &op.Username, &op.Email, &op.Role, &op.Active, &op.CreatedAt); err == nil {
					page.Operators = append(page.Operators, op)
				}
			}
		}
	}

	s.render(w, "settings.html", page)
}

func (s *Server) saveLetterType(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	name := strings.TrimSpace(r.FormValue("name"))
	requirements, err := parseLines(r.FormValue("requirements"), 20, 160, true)
	if !letterCodePattern.MatchString(code) || utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 120 || err != nil {
		http.Redirect(w, r, "/settings?notice=invalid-letter#letters", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err = s.db.ExecContext(ctx, `INSERT INTO jenis_surat(kode,nama,syarat,aktif) VALUES ($1,$2,$3,$4) ON CONFLICT (kode) DO UPDATE SET nama=EXCLUDED.nama,syarat=EXCLUDED.syarat,aktif=EXCLUDED.aktif`, code, name, requirements, r.FormValue("aktif") == "on")
	if err != nil {
		http.Error(w, "Jenis surat belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/settings?notice=saved#letters", http.StatusSeeOther)
}

// deleteFAQ menghapus satu FAQ secara permanen. Hanya admin yang sudah masuk
// dan membawa token CSRF yang valid yang dapat memakainya. Menonaktifkan FAQ
// (tanpa menghapus) tetap dilakukan lewat kotak "Aktif" pada dialog Edit.
func (s *Server) deleteFAQ(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("id")), 10, 64)
	if err != nil || id < 1 {
		http.Redirect(w, r, "/settings?notice=invalid-faq#faq", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(ctx, `DELETE FROM faq WHERE id=$1`, id)
	if err != nil {
		http.Error(w, "FAQ belum dapat dihapus.", http.StatusServiceUnavailable)
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/settings?notice=deleted#faq", http.StatusSeeOther)
}

func (s *Server) saveFAQ(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, currentAdmin(r).CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	question := strings.TrimSpace(r.FormValue("question"))
	answer := strings.TrimSpace(r.FormValue("answer"))
	category := strings.TrimSpace(r.FormValue("category"))
	keywords, err := parseLines(r.FormValue("keywords"), 20, 60, false)
	if utf8.RuneCountInString(question) < 3 || utf8.RuneCountInString(question) > 300 || utf8.RuneCountInString(answer) < 3 || utf8.RuneCountInString(answer) > 2000 || utf8.RuneCountInString(category) > 80 || err != nil {
		http.Redirect(w, r, "/settings?notice=invalid-faq#faq", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	active := r.FormValue("aktif") == "on"
	if rawID := strings.TrimSpace(r.FormValue("id")); rawID != "" {
		id, parseErr := strconv.ParseInt(rawID, 10, 64)
		if parseErr != nil || id < 1 {
			http.Redirect(w, r, "/settings?notice=invalid-faq#faq", http.StatusSeeOther)
			return
		}
		result, err := s.db.ExecContext(ctx, `UPDATE faq SET pertanyaan=$1,kata_kunci=$2,jawaban=$3,kategori=$4,aktif=$5 WHERE id=$6`, question, keywords, answer, category, active, id)
		if err != nil {
			http.Error(w, "FAQ belum dapat disimpan.", http.StatusServiceUnavailable)
			return
		}
		if changed, _ := result.RowsAffected(); changed == 0 {
			http.NotFound(w, r)
			return
		}
	} else if _, err := s.db.ExecContext(ctx, `INSERT INTO faq(pertanyaan,kata_kunci,jawaban,kategori,aktif) VALUES ($1,$2,$3,$4,$5)`, question, keywords, answer, category, active); err != nil {
		http.Error(w, "FAQ belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/settings?notice=saved#faq", http.StatusSeeOther)
}

func parseLines(raw string, maxCount, maxLength int, allowEmpty bool) ([]string, error) {
	items := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		item := strings.TrimSpace(line)
		if item == "" {
			continue
		}
		if utf8.RuneCountInString(item) > maxLength {
			return nil, fmt.Errorf("item too long")
		}
		items = append(items, item)
		if len(items) > maxCount {
			return nil, fmt.Errorf("too many items")
		}
	}
	if !allowEmpty && len(items) == 0 {
		return nil, fmt.Errorf("at least one item is required")
	}
	return items, nil
}

// adminResetOperatorPassword memfasilitasi admin utama mengirimkan tautan reset kata sandi ke email operator.
func (s *Server) adminResetOperatorPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	user := currentAdmin(r)
	if user.Role != "admin" {
		http.Error(w, "Hanya Administrator Utama yang berhak mengatur ulang kata sandi operator.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, user.CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	targetID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("id")), 10, 64)
	if err != nil || targetID < 1 {
		http.Redirect(w, r, "/settings?notice=invalid-operator#users", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var targetUser struct {
		ID       int64
		Username string
		Email    sql.NullString
		Role     string
		Active   bool
	}
	err = s.db.QueryRowContext(ctx, `SELECT id, username, email, role, aktif FROM admin_user WHERE id=$1`, targetID).Scan(&targetUser.ID, &targetUser.Username, &targetUser.Email, &targetUser.Role, &targetUser.Active)
	if errors.Is(err, sql.ErrNoRows) || !targetUser.Active {
		http.Redirect(w, r, "/settings?notice=operator-not-found#users", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Error(w, "Gagal memproses data pengguna.", http.StatusServiceUnavailable)
		return
	}

	email := strings.TrimSpace(targetUser.Email.String)
	if email == "" {
		http.Redirect(w, r, "/settings?notice=operator-no-email#users", http.StatusSeeOther)
		return
	}

	rawToken, err := randomToken(32)
	if err != nil {
		http.Error(w, "Gagal membuat token pemulihan.", http.StatusInternalServerError)
		return
	}
	tokenHash := sha256.Sum256([]byte(rawToken))
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Layanan database bermasalah.", http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `INSERT INTO password_reset_token(admin_id, token_hash, expires_at) VALUES ($1, $2, $3)`, targetUser.ID, tokenHash[:], expiresAt); err != nil {
		http.Error(w, "Gagal menyimpan tautan reset.", http.StatusInternalServerError)
		return
	}

	logDetails := fmt.Sprintf("Admin '%s' mengirim tautan reset kata sandi ke email pengguna '%s' (%s)", user.Username, targetUser.Username, email)
	_, _ = tx.ExecContext(ctx, `INSERT INTO admin_audit_log(admin_id, action, target_user_id, details, ip_address) VALUES ($1, 'admin_reset_password_request', $2, $3, $4)`, user.ID, targetUser.ID, logDetails, remoteIP(r))

	if err := tx.Commit(); err != nil {
		http.Error(w, "Gagal menyelesaikan transaksi.", http.StatusServiceUnavailable)
		return
	}

	baseURL := s.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://" + r.Host
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, rawToken)

	go func(toEmail, toUser, link string) {
		if err := s.mailer.SendPasswordReset(toEmail, toUser, link); err != nil {
			log.Printf("[MAILER ERROR] Gagal mengirim email reset operator %s: %v", toEmail, err)
		}
	}(email, targetUser.Username, resetURL)

	http.Redirect(w, r, "/settings?notice=reset-link-sent#users", http.StatusSeeOther)
}

// saveOperator menambah operator baru atau memperbarui data akun operator (email, status aktif).
func (s *Server) saveOperator(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	user := currentAdmin(r)
	if user.Role != "admin" {
		http.Error(w, "Akses ditolak.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, user.CSRF) {
		http.Error(w, "Permintaan tidak valid.", http.StatusForbidden)
		return
	}
	idStr := strings.TrimSpace(r.FormValue("id"))
	username := strings.TrimSpace(r.FormValue("username"))
	rawEmail := strings.TrimSpace(r.FormValue("email"))
	role := strings.TrimSpace(r.FormValue("role"))
	active := r.FormValue("aktif") == "on"

	if role != "operator" && role != "admin" {
		role = "operator"
	}
	var cleanEmail string
	if rawEmail != "" {
		addr, err := mail.ParseAddress(rawEmail)
		if err != nil {
			http.Redirect(w, r, "/settings?notice=invalid-operator-email#users", http.StatusSeeOther)
			return
		}
		cleanEmail = strings.ToLower(addr.Address)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			http.Redirect(w, r, "/settings?notice=invalid-operator#users", http.StatusSeeOther)
			return
		}
		if id == user.ID && !active {
			active = true // Admin tidak boleh menonaktifkan dirinya sendiri
		}
		_, err = s.db.ExecContext(ctx, `UPDATE admin_user SET email=NULLIF($1, ''), aktif=$2 WHERE id=$3`, cleanEmail, active, id)
		if err != nil {
			http.Redirect(w, r, "/settings?notice=duplicate-operator-email#users", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/settings?notice=operator-saved#users", http.StatusSeeOther)
		return
	}

	if username == "" || len(username) > 80 {
		http.Redirect(w, r, "/settings?notice=invalid-operator-username#users", http.StatusSeeOther)
		return
	}
	password := r.FormValue("password")
	if utf8.RuneCountInString(password) < 12 {
		http.Redirect(w, r, "/settings?notice=operator-password-too-short#users", http.StatusSeeOther)
		return
	}
	passHash, err := security.HashPassword(password)
	if err != nil {
		http.Error(w, "Gagal mengolah kata sandi.", http.StatusInternalServerError)
		return
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO admin_user(username, password_hash, role, email, aktif) VALUES ($1, $2, $3, NULLIF($4, ''), $5)`, username, passHash, role, cleanEmail, active)
	if err != nil {
		http.Redirect(w, r, "/settings?notice=duplicate-operator#users", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/settings?notice=operator-saved#users", http.StatusSeeOther)
}
