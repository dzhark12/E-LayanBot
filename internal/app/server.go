package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elayan.local/desa/config"
	"elayan.local/desa/internal/storage"
)

type Server struct {
	db        *sql.DB
	cfg       config.Config
	files     *storage.Local
	templates *template.Template
}

type adminUser struct {
	ID       int64
	Username string
	Role     string
	CSRF     string
}

type contextKey string

const adminContextKey contextKey = "elayan-admin"

func New(db *sql.DB, cfg config.Config, files *storage.Local) (*Server, error) {
	tmpl, err := template.New("pages").Funcs(template.FuncMap{
		"date": func(t time.Time) string { return t.Format("02 Jan 2006, 15:04") },
		"statusClass": func(status string) string {
			switch status {
			case "Menunggu":
				return "waiting"
			case "Dikerjakan":
				return "working"
			case "Selesai":
				return "done"
			case "Perlu Perbaikan":
				return "revision"
			default:
				return "rejected"
			}
		},
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"divFloat":  func(a int64, b int64) float64 { return float64(a) / float64(b) },
		"joinLines": func(lines []string) string { return strings.Join(lines, "\n") },
	}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{db: db, cfg: cfg, files: files, templates: tmpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.Handle("POST /logout", s.requireAdmin(http.HandlerFunc(s.logout)))
	mux.Handle("GET /", s.requireAdmin(http.HandlerFunc(s.dashboard)))
	mux.Handle("GET /admin/requests/version", s.requireAdmin(http.HandlerFunc(s.requestVersion)))
	mux.Handle("GET /settings", s.requireAdmin(http.HandlerFunc(s.settings)))
	mux.Handle("POST /settings/letter", s.requireAdmin(http.HandlerFunc(s.saveLetterType)))
	mux.Handle("POST /settings/faq", s.requireAdmin(http.HandlerFunc(s.saveFAQ)))
	mux.Handle("POST /settings/faq/delete", s.requireAdmin(http.HandlerFunc(s.deleteFAQ)))
	mux.Handle("GET /requests/{id}", s.requireAdmin(http.HandlerFunc(s.requestDetail)))
	mux.Handle("POST /requests/{id}/status", s.requireAdmin(http.HandlerFunc(s.updateStatus)))
	mux.Handle("POST /requests/{id}/identity", s.requireAdmin(http.HandlerFunc(s.updateIdentity)))
	mux.Handle("GET /requests/{id}/letter.pdf", s.requireAdmin(http.HandlerFunc(s.letterPDF)))
	mux.Handle("POST /requests/{id}/letter.pdf", s.requireAdmin(http.HandlerFunc(s.regenerateLetterPDF)))
	mux.Handle("GET /attachments/{id}", s.requireAdmin(http.HandlerFunc(s.attachment)))
	mux.HandleFunc("POST /api/v1/submissions", s.botAuth(s.createSubmission))
	mux.HandleFunc("POST /api/v1/status", s.botAuth(s.lookupStatus))
	mux.HandleFunc("GET /api/v1/letter-types", s.botAuth(s.listLetterTypes))
	mux.HandleFunc("GET /api/v1/faq", s.botAuth(s.lookupFAQ))

	return s.securityHeaders(mux)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; form-action 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'")
		if s.cfg.CookieSecure {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "service unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Halaman tidak dapat ditampilkan.", http.StatusInternalServerError)
	}
}

// renderStatus menampilkan template dengan kode status HTTP tertentu (misalnya
// 429 atau 503). Template dirender ke buffer dulu agar kode status tidak
// terkirim sebelum template dipastikan berhasil.
func (s *Server) renderStatus(w http.ResponseWriter, status int, name string, data any) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "Halaman tidak dapat ditampilkan.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	var body apiError
	body.Error.Code, body.Error.Message = code, message
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func pageInt(value string) int {
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 {
		return 1
	}
	return page
}
