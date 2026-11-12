package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

import "elayan.local/desa/internal/messenger"

type notification struct {
	ID   int64
	JID  string
	Code string
}

func RunNotifications(ctx context.Context, db *sql.DB, wa messenger.Messenger, logger *log.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		processNotification(ctx, db, wa, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func processNotification(ctx context.Context, db *sql.DB, wa messenger.Messenger, logger *log.Logger) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Printf("notification queue unavailable: %v", err)
		return
	}
	defer tx.Rollback()
	var item notification
	err = tx.QueryRowContext(ctx, `SELECT n.id,p.wa_jid,p.kode_unik FROM notifikasi_log n JOIN permohonan p ON p.id=n.permohonan_id WHERE n.status_terkirim='Selesai' AND n.sent_at IS NULL AND n.attempt_count < 5 AND n.next_attempt_at <= now() AND (n.claimed_until IS NULL OR n.claimed_until < now()) ORDER BY n.waktu LIMIT 1 FOR UPDATE OF n SKIP LOCKED`).Scan(&item.ID, &item.JID, &item.Code)
	if err == sql.ErrNoRows {
		return
	}
	if err != nil {
		logger.Printf("notification queue read failed: %v", err)
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE notifikasi_log SET attempt_count=attempt_count+1,claimed_until=now()+interval '1 minute' WHERE id=$1`, item.ID); err != nil {
		logger.Printf("notification queue claim failed: %v", err)
		return
	}
	if err := tx.Commit(); err != nil {
		logger.Printf("notification queue claim failed: %v", err)
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err = wa.SendText(sendCtx, item.JID, fmt.Sprintf("Pengajuan surat dengan kode %s telah selesai. Silakan hubungi perangkat desa untuk pengambilan surat.", item.Code))
	cancel()
	if err != nil {
		_, updateErr := db.ExecContext(ctx, `UPDATE notifikasi_log SET error='Pengiriman gagal',next_attempt_at=now()+interval '5 minutes',claimed_until=NULL,waktu=now() WHERE id=$1`, item.ID)
		if updateErr != nil {
			logger.Printf("notification failure could not be recorded: %v", updateErr)
		}
		logger.Printf("notification delivery failed for request %s", item.Code)
		return
	}
	if _, err := db.ExecContext(ctx, `UPDATE notifikasi_log SET sent_at=now(),error='',claimed_until=NULL,waktu=now() WHERE id=$1`, item.ID); err != nil {
		logger.Printf("notification delivery could not be recorded for request %s: %v", item.Code, err)
	}
}
