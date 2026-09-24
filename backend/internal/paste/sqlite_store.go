package paste

import (
	"context"
	"crypto/subtle"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nightiz/pastebin-clone/backend/migrations"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates and initializes a SQLite backed Paste Store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create database directory: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Configure connection pragmas for performance & reliability
	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	// Apply migration schema
	if _, err := db.Exec(migrations.InitSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply initial migration: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Create(ctx context.Context, p *Paste, editTokenHash string) error {
	query := `
		INSERT INTO pastes (id, title, content, language, created_at, expires_at, burn_after_read, views, edit_token_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAtStr *string
	if p.ExpiresAt != nil {
		val := p.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAtStr = &val
	}

	burnVal := 0
	if p.BurnAfterRead {
		burnVal = 1
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
		p.ID,
		p.Title,
		p.Content,
		p.Language,
		p.CreatedAt.UTC().Format(time.RFC3339),
		expiresAtStr,
		burnVal,
		p.Views,
		editTokenHash,
	)
	if err != nil {
		return fmt.Errorf("failed to insert paste: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, id string) (*Paste, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		SELECT id, title, content, language, created_at, expires_at, burn_after_read, views
		FROM pastes
		WHERE id = ?
	`

	var (
		p            Paste
		createdAtStr string
		expiresAtStr sql.NullString
		burnVal      int
	)

	err = tx.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.Title,
		&p.Content,
		&p.Language,
		&createdAtStr,
		&expiresAtStr,
		&burnVal,
		&p.Views,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query paste: %w", err)
	}

	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	p.BurnAfterRead = burnVal == 1

	if expiresAtStr.Valid {
		t, parseErr := time.Parse(time.RFC3339, expiresAtStr.String)
		if parseErr == nil {
			p.ExpiresAt = &t
			if time.Now().UTC().After(t) {
				// Lazy purge on read
				_, _ = tx.ExecContext(ctx, `DELETE FROM pastes WHERE id = ?`, id)
				_ = tx.Commit()
				return nil, ErrNotFound
			}
		}
	}

	if p.BurnAfterRead {
		// Burn after read: delete row after single successful read
		if _, err := tx.ExecContext(ctx, `DELETE FROM pastes WHERE id = ?`, id); err != nil {
			return nil, fmt.Errorf("failed to burn paste: %w", err)
		}
		p.Views++
	} else {
		// Increment views
		if _, err := tx.ExecContext(ctx, `UPDATE pastes SET views = views + 1 WHERE id = ?`, id); err != nil {
			return nil, fmt.Errorf("failed to increment view count: %w", err)
		}
		p.Views++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit read transaction: %w", err)
	}

	return &p, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM pastes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete paste: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) VerifyEditToken(ctx context.Context, id string, tokenHash string) (bool, error) {
	var storedHash string
	err := s.db.QueryRowContext(ctx, `SELECT edit_token_hash FROM pastes WHERE id = ?`, id).Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("failed to query edit token hash: %w", err)
	}

	// Constant time comparison to prevent timing leaks
	matches := subtle.ConstantTimeCompare([]byte(storedHash), []byte(tokenHash)) == 1
	return matches, nil
}

func (s *SQLiteStore) PurgeExpired(ctx context.Context) (int64, error) {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `DELETE FROM pastes WHERE expires_at IS NOT NULL AND expires_at <= ?`, nowStr)
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired pastes: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
