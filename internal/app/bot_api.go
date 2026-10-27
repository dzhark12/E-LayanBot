package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type letterTypeResponse struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Requirements []string `json:"requirements"`
}

func (s *Server) listLetterTypes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT kode,nama,array_to_json(syarat)::text FROM jenis_surat WHERE aktif=TRUE ORDER BY kode`)
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "jenis surat belum dapat dimuat")
		return
	}
	defer rows.Close()
	result := make([]letterTypeResponse, 0, 5)
	for rows.Next() {
		var item letterTypeResponse
		var requirements []byte
		if err := rows.Scan(&item.Code, &item.Name, &requirements); err != nil || json.Unmarshal(requirements, &item.Requirements) != nil {
			writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "jenis surat belum dapat dimuat")
			return
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "jenis surat belum dapat dimuat")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) lookupFAQ(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) < 3 || utf8.RuneCountInString(query) > 300 {
		writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", "pertanyaan tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	var answer string
	err := s.db.QueryRowContext(ctx, `SELECT jawaban FROM faq WHERE aktif=TRUE AND EXISTS (SELECT 1 FROM unnest(kata_kunci) AS keywords(keyword) WHERE length(keywords.keyword) >= 3 AND lower($1) LIKE '%' || lower(keywords.keyword) || '%') ORDER BY id LIMIT 1`, query).Scan(&answer)
	if err == sql.ErrNoRows {
		writeAPIError(w, http.StatusNotFound, "FAQ_NOT_FOUND", "informasi tidak ditemukan")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "informasi belum dapat dimuat")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"answer": answer})
}
