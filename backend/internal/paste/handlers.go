package paste

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/nightiz/pastebin-clone/backend/internal/apierr"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/pastes
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePasteRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			apierr.WriteErr(w, apierr.PayloadTooLarge("request payload exceeds maximum allowed size"))
			return
		}
		apierr.WriteErr(w, apierr.ValidationError("invalid JSON request body"))
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyContent):
			apierr.WriteErr(w, apierr.ValidationError("content is required and cannot be empty"))
		case errors.Is(err, ErrContentTooLarge):
			apierr.WriteErr(w, apierr.PayloadTooLarge("paste content exceeds maximum size"))
		case errors.Is(err, ErrInvalidExpiry):
			apierr.WriteErr(w, apierr.ValidationError("expiry duration must be non-negative"))
		default:
			apierr.WriteErr(w, apierr.Internal("failed to create paste"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// Get handles GET /api/pastes/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierr.WriteErr(w, apierr.NotFound("paste not found"))
		return
	}

	p, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.WriteErr(w, apierr.NotFound("paste not found"))
			return
		}
		apierr.WriteErr(w, apierr.Internal("failed to retrieve paste"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

// GetRaw handles GET /api/pastes/{id}/raw
func (h *Handler) GetRaw(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierr.WriteErr(w, apierr.NotFound("paste not found"))
		return
	}

	p, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.WriteErr(w, apierr.NotFound("paste not found"))
			return
		}
		apierr.WriteErr(w, apierr.Internal("failed to retrieve paste"))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, p.Content)
}

// Delete handles DELETE /api/pastes/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierr.WriteErr(w, apierr.NotFound("paste not found"))
		return
	}

	token := extractEditToken(r)
	if token == "" {
		apierr.WriteErr(w, apierr.Unauthorized("missing edit token"))
		return
	}

	err := h.service.Delete(r.Context(), id, token)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			apierr.WriteErr(w, apierr.NotFound("paste not found"))
		case errors.Is(err, ErrInvalidEditToken):
			apierr.WriteErr(w, apierr.Unauthorized("invalid edit token"))
		default:
			apierr.WriteErr(w, apierr.Internal("failed to delete paste"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "paste deleted"})
}

// extractEditToken checks header X-Edit-Token, Authorization Bearer, query params, or JSON body
func extractEditToken(r *http.Request) string {
	if tok := r.Header.Get("X-Edit-Token"); tok != "" {
		return strings.TrimSpace(tok)
	}

	if auth := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}

	if tok := r.URL.Query().Get("token"); tok != "" {
		return strings.TrimSpace(tok)
	}

	if tok := r.URL.Query().Get("edit_token"); tok != "" {
		return strings.TrimSpace(tok)
	}

	// Try reading JSON body if available
	var body struct {
		EditToken string `json:"edit_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.EditToken != "" {
		return strings.TrimSpace(body.EditToken)
	}

	return ""
}
