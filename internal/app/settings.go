package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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

type settingsPage struct {
	User    adminUser
	CSRF    string
	Letters []letterSetting
	FAQs    []faqSetting
	Notice  string
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
		http.Redirect(w, r, "/settings?notice=invalid-letter", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err = s.db.ExecContext(ctx, `INSERT INTO jenis_surat(kode,nama,syarat,aktif) VALUES ($1,$2,$3,$4) ON CONFLICT (kode) DO UPDATE SET nama=EXCLUDED.nama,syarat=EXCLUDED.syarat,aktif=EXCLUDED.aktif`, code, name, requirements, r.FormValue("aktif") == "on")
	if err != nil {
		http.Error(w, "Jenis surat belum dapat disimpan.", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/settings?notice=saved", http.StatusSeeOther)
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
		http.Redirect(w, r, "/settings?notice=invalid-faq", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	active := r.FormValue("aktif") == "on"
	if rawID := strings.TrimSpace(r.FormValue("id")); rawID != "" {
		id, parseErr := strconv.ParseInt(rawID, 10, 64)
		if parseErr != nil || id < 1 {
			http.Redirect(w, r, "/settings?notice=invalid-faq", http.StatusSeeOther)
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
	http.Redirect(w, r, "/settings?notice=saved", http.StatusSeeOther)
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
