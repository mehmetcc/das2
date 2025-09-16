package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/mehmetcc/das2/internal/httpx"
	"github.com/mehmetcc/das2/internal/person"
	"go.uber.org/zap"
)

type AuthenticationHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Routes() chi.Router
}

type authenticationHandler struct {
	logger      *zap.Logger
	authService AuthService
	validator   *validator.Validate
}

func NewAuthenticationHandler(authService AuthService, l *zap.Logger) AuthenticationHandler {
	v := validator.New(validator.WithRequiredStructEnabled())
	return &authenticationHandler{
		logger:      l,
		authService: authService,
		validator:   v,
	}
}

func (a *authenticationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", a.Register)
	return r
}

func (a *authenticationHandler) Register(w http.ResponseWriter, r *http.Request) {
	/** common checks for all endpoints **/
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, httpx.ErrorResponse[any]{
			Code:    httpx.ErrUnsupportedMedia,
			Message: "Content-Type must be application/json",
		})
		return
	}

	/** unmarshal & validate here */
	var req registerPersonRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		a.logger.Warn("failed to decode register request body", zap.Error(err))
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrorResponse[any]{
			Code:    httpx.ErrInvalidJSON,
			Message: "invalid request body",
		})
		return
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF { // check if there's any trailing data
		a.logger.Warn("trailing data after JSON body", zap.Error(err))
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrorResponse[any]{
			Code:    httpx.ErrInvalidJSON,
			Message: "request body must contain a single JSON object",
		})
		return
	}

	if err := a.validator.Struct(req); err != nil {
		a.logger.Warn("register validation failed", zap.Error(err))
		fields := httpx.ValidationDetails(err)
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.ErrorResponse[[]httpx.FieldError]{
			Code:    httpx.ErrValidationFailed,
			Message: "validation failed",
			Details: fields,
		})
		return
	}

	/** Business logic */
	id, err := a.authService.Register(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		a.logger.Warn("failed to register user", zap.Error(err))
		switch err {
		case person.ErrDuplicateEmail:
			a.logger.Debug("duplicate email", zap.String("email", req.Email))
			httpx.WriteError(w, http.StatusConflict, httpx.ErrorResponse[any]{
				Code:    httpx.ErrConflict,
				Message: "email already exists",
			})
		case person.ErrDuplicateUsername:
			a.logger.Debug("duplicate username", zap.String("username", req.Username))
			httpx.WriteError(w, http.StatusConflict, httpx.ErrorResponse[any]{
				Code:    httpx.ErrConflict,
				Message: "username already exists",
			})
		default:
			a.logger.Error("internal server error", zap.Error(err))
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrorResponse[any]{
				Code:    httpx.ErrInternal,
				Message: "internal server error",
			})
		}
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, registerPersonResponse{
		PublicID: string(id),
	})
}

type registerPersonRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Username string `json:"username" validate:"required,min=8,max=32"`
	Password string `json:"password" validate:"required,min=8,,max=72,alphanum"`
}

type registerPersonResponse struct {
	PublicID string `json:"public_id"`
}
