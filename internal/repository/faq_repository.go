package repository

import (
	"context"
	"database/sql"
)

type FAQEntry struct{
	Question string
	Answer string
}

func ListActiveFAQ(ctx context.Context, db *sql.DB, limit int) ([]FAQEntry, error){
	rows, err := db.QueryContext(ctx, `SELECT pertanyaan,jawaban FROM faq WHERE aktif=TRUE ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FAQEntry
	for rows.Next() {
		var e FAQEntry
		if err := rows.Scan(&e.Question, &e.Answer); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}