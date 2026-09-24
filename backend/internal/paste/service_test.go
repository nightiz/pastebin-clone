package paste

import (
	"context"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestGenerateBase62ID(t *testing.T) {
	for i := 0; i < 50; i++ {
		id, err := GenerateBase62ID(8)
		if err != nil {
			t.Fatalf("GenerateBase62ID error: %v", err)
		}
		if len(id) != 8 {
			t.Errorf("got length %d, want 8", len(id))
		}
		for _, c := range id {
			if !strings.ContainsRune(base62Chars, c) {
				t.Errorf("unexpected character %c in id %s", c, id)
			}
		}
	}
}

func TestService_CreateAndGet(t *testing.T) {
	store := newTestStore(t)
	svc := NewService(store, 524288, 0)
	ctx := context.Background()

	// 1. Validation: empty content
	_, err := svc.Create(ctx, CreatePasteRequest{Content: "   "})
	if err != ErrEmptyContent {
		t.Errorf("expected ErrEmptyContent, got %v", err)
	}

	// 2. Validation: content too large
	smallSvc := NewService(store, 5, 0)
	_, err = smallSvc.Create(ctx, CreatePasteRequest{Content: "too long content"})
	if err != ErrContentTooLarge {
		t.Errorf("expected ErrContentTooLarge, got %v", err)
	}

	// 3. Normal create
	resp, err := svc.Create(ctx, CreatePasteRequest{
		Title:    "Hello World",
		Content:  "fmt.Println(\"hi\")",
		Language: "go",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == "" || resp.EditToken == "" {
		t.Fatalf("missing ID or EditToken in response: %+v", resp)
	}

	// 4. Retrieve
	p, err := svc.Get(ctx, resp.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if p.Title != "Hello World" || p.Content != "fmt.Println(\"hi\")" || p.Views != 1 {
		t.Errorf("unexpected paste fields: %+v", p)
	}
}

func TestService_BurnAfterRead(t *testing.T) {
	store := newTestStore(t)
	svc := NewService(store, 524288, 0)
	ctx := context.Background()

	resp, err := svc.Create(ctx, CreatePasteRequest{
		Title:         "Secret",
		Content:       "Nuclear codes",
		BurnAfterRead: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// First read succeeds
	p, err := svc.Get(ctx, resp.ID)
	if err != nil {
		t.Fatalf("first Get failed: %v", err)
	}
	if p.Content != "Nuclear codes" {
		t.Errorf("got %q, want 'Nuclear codes'", p.Content)
	}

	// Second read returns ErrNotFound
	_, err = svc.Get(ctx, resp.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound on second read, got %v", err)
	}
}

func TestService_Expiry(t *testing.T) {
	store := newTestStore(t)
	svc := NewService(store, 524288, 0)
	ctx := context.Background()

	// Pastes with 1 second expiry
	expSec := 1
	resp, err := svc.Create(ctx, CreatePasteRequest{
		Content:          "Will expire soon",
		ExpiresInSeconds: &expSec,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Immediate read succeeds
	_, err = svc.Get(ctx, resp.ID)
	if err != nil {
		t.Fatalf("immediate Get failed: %v", err)
	}

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// Read after expiry fails
	_, err = svc.Get(ctx, resp.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after expiration, got %v", err)
	}
}

func TestService_Delete(t *testing.T) {
	store := newTestStore(t)
	svc := NewService(store, 524288, 0)
	ctx := context.Background()

	resp, err := svc.Create(ctx, CreatePasteRequest{
		Content: "Delete me",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete with wrong token fails
	err = svc.Delete(ctx, resp.ID, "wrong-token")
	if err != ErrInvalidEditToken {
		t.Errorf("expected ErrInvalidEditToken, got %v", err)
	}

	// Delete with correct token succeeds
	err = svc.Delete(ctx, resp.ID, resp.EditToken)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Fetch after delete fails
	_, err = svc.Get(ctx, resp.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}
