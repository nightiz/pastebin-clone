package paste

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEmptyContent     = errors.New("paste content cannot be empty")
	ErrContentTooLarge  = errors.New("paste content exceeds maximum size")
	ErrInvalidExpiry    = errors.New("expiry duration must be non-negative")
	ErrInvalidEditToken = errors.New("invalid or missing edit token")
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateBase62ID generates a cryptographically secure random base62 string of given length
// with no modulo bias.
func GenerateBase62ID(length int) (string, error) {
	bytes := make([]byte, length)
	buf := make([]byte, 1)
	for i := 0; i < length; {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate random id: %w", err)
		}
		// 62 * 4 = 248. Discard >= 248 to guarantee uniform distribution
		if buf[0] < 248 {
			bytes[i] = base62Chars[buf[0]%62]
			i++
		}
	}
	return string(bytes), nil
}

// GenerateEditToken generates a cryptographically secure 32-byte hex token.
func GenerateEditToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate edit token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 hash of the token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type Service struct {
	store                Store
	maxPasteBytes        int64
	defaultExpirySeconds int
}

func NewService(store Store, maxPasteBytes int64, defaultExpirySeconds int) *Service {
	return &Service{
		store:                store,
		maxPasteBytes:        maxPasteBytes,
		defaultExpirySeconds: defaultExpirySeconds,
	}
}

func (s *Service) Create(ctx context.Context, req CreatePasteRequest) (*CreatePasteResponse, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, ErrEmptyContent
	}
	if int64(len(req.Content)) > s.maxPasteBytes {
		return nil, ErrContentTooLarge
	}

	expirySec := s.defaultExpirySeconds
	if req.ExpiresInSeconds != nil {
		if *req.ExpiresInSeconds < 0 {
			return nil, ErrInvalidExpiry
		}
		expirySec = *req.ExpiresInSeconds
	}

	id, err := GenerateBase62ID(8)
	if err != nil {
		return nil, err
	}

	editToken, err := GenerateEditToken()
	if err != nil {
		return nil, err
	}
	tokenHash := HashToken(editToken)

	now := time.Now().UTC()
	var expiresAt *time.Time
	if expirySec > 0 {
		t := now.Add(time.Duration(expirySec) * time.Second)
		expiresAt = &t
	}

	p := &Paste{
		ID:            id,
		Title:         req.Title,
		Content:       req.Content,
		Language:      req.Language,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
		BurnAfterRead: req.BurnAfterRead,
		Views:         0,
	}

	if err := s.store.Create(ctx, p, tokenHash); err != nil {
		return nil, fmt.Errorf("service failed to store paste: %w", err)
	}

	return &CreatePasteResponse{
		ID:            p.ID,
		Title:         p.Title,
		Content:       p.Content,
		Language:      p.Language,
		CreatedAt:     p.CreatedAt,
		ExpiresAt:     p.ExpiresAt,
		BurnAfterRead: p.BurnAfterRead,
		Views:         p.Views,
		EditToken:     editToken,
	}, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Paste, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	return s.store.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string, editToken string) error {
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	if strings.TrimSpace(editToken) == "" {
		return ErrInvalidEditToken
	}

	tokenHash := HashToken(editToken)
	valid, err := s.store.VerifyEditToken(ctx, id, tokenHash)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidEditToken
	}

	return s.store.Delete(ctx, id)
}

func (s *Service) PurgeExpired(ctx context.Context) (int64, error) {
	return s.store.PurgeExpired(ctx)
}
