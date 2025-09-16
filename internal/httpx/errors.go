package httpx

import "github.com/go-playground/validator/v10"

type ErrorCode string

const (
	ErrInvalidJSON      ErrorCode = "invalid_json"
	ErrUnsupportedMedia ErrorCode = "unsupported_media_type"
	ErrValidationFailed ErrorCode = "validation_failed"
	ErrUnauthorized     ErrorCode = "unauthorized"
	ErrNotFound         ErrorCode = "not_found"
	ErrConflict         ErrorCode = "conflict"
	ErrInternal         ErrorCode = "internal_error"
)

type FieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Param string `json:"param,omitempty"`
}

type ErrorResponse[T any] struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details T         `json:"details,omitempty"`
}

func ValidationDetails(err error) []FieldError {
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return []FieldError{{Field: "", Rule: "invalid", Param: err.Error()}}
	}
	out := make([]FieldError, 0, len(verrs))
	for _, e := range verrs {
		out = append(out, FieldError{
			Field: e.Field(),
			Rule:  e.Tag(),
			Param: e.Param(),
		})
	}
	return out
}
