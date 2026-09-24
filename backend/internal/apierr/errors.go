package apierr

import (
	"encoding/json"
	"net/http"
)

const (
	CodeNotFound        = "NOT_FOUND"
	CodeValidationError = "VALIDATION_ERROR"
	CodePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	CodeRateLimited     = "RATE_LIMITED"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeInternal        = "INTERNAL"
)

type APIError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}

type ErrorResponse struct {
	Error *APIError `json:"error"`
}

func New(status int, code, message string) *APIError {
	return &APIError{
		HTTPStatus: status,
		Code:       code,
		Message:    message,
	}
}

func NotFound(msg string) *APIError {
	if msg == "" {
		msg = "resource not found"
	}
	return New(http.StatusNotFound, CodeNotFound, msg)
}

func ValidationError(msg string) *APIError {
	return New(http.StatusBadRequest, CodeValidationError, msg)
}

func PayloadTooLarge(msg string) *APIError {
	if msg == "" {
		msg = "request payload too large"
	}
	return New(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, msg)
}

func RateLimited(msg string) *APIError {
	if msg == "" {
		msg = "too many requests, please slow down"
	}
	return New(http.StatusTooManyRequests, CodeRateLimited, msg)
}

func Unauthorized(msg string) *APIError {
	if msg == "" {
		msg = "unauthorized: invalid or missing token"
	}
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Internal(msg string) *APIError {
	if msg == "" {
		msg = "an internal server error occurred"
	}
	return New(http.StatusInternalServerError, CodeInternal, msg)
}

// Write writes an API error as JSON with appropriate HTTP status code
func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: &APIError{Code: code, Message: message},
	})
}

// WriteErr writes an existing APIError to response
func WriteErr(w http.ResponseWriter, err *APIError) {
	if err == nil {
		err = Internal("")
	}
	Write(w, err.HTTPStatus, err.Code, err.Message)
}
