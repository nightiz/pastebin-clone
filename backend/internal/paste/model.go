package paste

import "time"

type Paste struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	Language      string     `json:"language"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	BurnAfterRead bool       `json:"burn_after_read"`
	Views         int        `json:"views"`
}

type CreatePasteRequest struct {
	Content          string `json:"content"`
	Title            string `json:"title"`
	Language         string `json:"language"`
	ExpiresInSeconds *int   `json:"expires_in_seconds"`
	BurnAfterRead    bool   `json:"burn_after_read"`
}

type CreatePasteResponse struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	Language      string     `json:"language"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	BurnAfterRead bool       `json:"burn_after_read"`
	Views         int        `json:"views"`
	EditToken     string     `json:"edit_token"`
}
