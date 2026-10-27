package app

import (
	"context"
	"log"
	"time"
)

func (s *Server) RunRetention(ctx context.Context, logger *log.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		s.purgeExpired(ctx, logger)
		if _, err := s.db.ExecContext(ctx, `DELETE FROM admin_session WHERE expires_at < now()`); err != nil && ctx.Err() == nil {
			logger.Printf("expired session cleanup failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) purgeExpired(ctx context.Context, logger *log.Logger) {
	rows, err := s.db.QueryContext(ctx, `SELECT l.id::text, l.storage_key FROM lampiran l JOIN permohonan p ON p.id=l.permohonan_id WHERE p.status='Selesai' AND p.selesai_pada <= now() - ($1 * interval '1 day') ORDER BY p.selesai_pada LIMIT 500`, s.cfg.RetentionDays)
	if err != nil {
		logger.Printf("lampiran retention query failed: %v", err)
		return
	}
	type expiredFile struct{ id, key string }
	var expired []expiredFile
	for rows.Next() {
		var file expiredFile
		if err := rows.Scan(&file.id, &file.key); err != nil {
			rows.Close()
			logger.Printf("lampiran retention scan failed: %v", err)
			return
		}
		expired = append(expired, file)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		logger.Printf("lampiran retention query failed: %v", err)
		return
	}
	rows.Close()
	for _, file := range expired {
		if err := s.files.Remove(file.key); err != nil {
			logger.Printf("lampiran retention delete failed: %v", err)
			continue
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM lampiran WHERE id=$1`, file.id); err != nil {
			logger.Printf("lampiran retention metadata delete failed: %v", err)
		}
	}
}
