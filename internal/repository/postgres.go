package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"elayan.local/desa/internal/security"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".sql")
		var applied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version = $1)`, version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sqlBytes, err := migrations.ReadFile(file)
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		for _, statement := range strings.Split(string(sqlBytes), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %s: %w", version, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migration(version) VALUES ($1)`, version); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func Seed(ctx context.Context, db *sql.DB, username, password, email string) error {
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	cleanEmail := strings.TrimSpace(email)
	if _, err := db.ExecContext(ctx, `INSERT INTO admin_user(username, password_hash, role, email) VALUES ($1, $2, 'admin', NULLIF($3, '')) ON CONFLICT (username) DO NOTHING`, username, passwordHash, cleanEmail); err != nil {
		return err
	}
	if cleanEmail != "" {
		_, _ = db.ExecContext(ctx, `UPDATE admin_user SET email=$2 WHERE username=$1 AND (email IS NULL OR email='')`, username, cleanEmail)
	}
	for _, letter := range []struct {
		code, name   string
		requirements []string
	}{
		{"DM", "Surat Keterangan Domisili", []string{"KTP pemohon", "KK", "Surat pengantar RT"}},
		{"SU", "Surat Keterangan Usaha", []string{"KTP pemohon", "KK", "Surat pengantar RT"}},
		{"TM", "Surat Keterangan Tidak Mampu", []string{"KTP pemohon", "KK", "Surat pengantar RT"}},
		{"KM", "Surat Keterangan Kematian", []string{"KTP almarhum", "KK almarhum", "Surat pengantar RT", "KTP pelapor"}},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO jenis_surat(kode, nama, syarat) VALUES ($1, $2, $3) ON CONFLICT (kode) DO NOTHING`,
			letter.code, letter.name, letter.requirements); err != nil {
			return err
		}
	}
	return nil
}
