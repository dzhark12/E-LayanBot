package repository

import (
	"context"
	"database/sql"

	"elayan.local/desa/internal/domain"
)

func LoadBotSession(ctx context.Context, db *sql.DB, jid string) (domain.SesiBot, error) {
	var session domain.SesiBot
	err := db.QueryRowContext(ctx, `SELECT jid, flow, step, data, updated_at FROM sesi_bot WHERE jid=$1`, jid).Scan(&session.JID, &session.Flow, &session.Step, &session.Data, &session.DiubahPada)
	return session, err
}

func SaveBotSession(ctx context.Context, db *sql.DB, session domain.SesiBot) error {
	_, err := db.ExecContext(ctx, `INSERT INTO sesi_bot(jid,flow,step,data,updated_at) VALUES ($1,$2,$3,$4::jsonb,now()) ON CONFLICT (jid) DO UPDATE SET flow=EXCLUDED.flow, step=EXCLUDED.step, data=EXCLUDED.data, updated_at=now()`, session.JID, session.Flow, session.Step, string(session.Data))
	return err
}

func DeleteBotSession(ctx context.Context, db *sql.DB, jid string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sesi_bot WHERE jid=$1`, jid)
	return err
}

func ExpireBotSessions(ctx context.Context, db *sql.DB) ([]domain.SesiBot, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT jid,flow,step,data,updated_at FROM sesi_bot WHERE updated_at <= now() - interval '24 hours' FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	var expired []domain.SesiBot
	for rows.Next() {
		var session domain.SesiBot
		if err := rows.Scan(&session.JID, &session.Flow, &session.Step, &session.Data, &session.DiubahPada); err != nil {
			rows.Close()
			return nil, err
		}
		expired = append(expired, session)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if _, err := tx.ExecContext(ctx, `DELETE FROM sesi_bot WHERE updated_at <= now() - interval '24 hours'`); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return expired, nil
}
