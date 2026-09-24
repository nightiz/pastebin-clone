package paste

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("paste not found")
)

// Store defines persistence operations for pastes.
//
// Purge Strategy:
// We use a dual purge approach:
// 1. Lazy check-on-read: When Get() fetches a paste, it checks whether expires_at has passed.
//    If expired, it is immediately deleted and ErrNotFound is returned.
// 2. Periodic background purge: A background worker periodically runs PurgeExpired() to sweep
//    pastes that expired without being accessed, preventing unbounded database growth.
// ponytail: dual purge (lazy check-on-read + background ticker) keeps SQLite lean without requiring external cron.
type Store interface {
	Create(ctx context.Context, p *Paste, editTokenHash string) error
	Get(ctx context.Context, id string) (*Paste, error)
	Delete(ctx context.Context, id string) error
	VerifyEditToken(ctx context.Context, id string, tokenHash string) (bool, error)
	PurgeExpired(ctx context.Context) (int64, error)
	Close() error
}
