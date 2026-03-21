// Package handler contains HTTP handlers for user endpoints.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/golikeit/domain"
)

//go:generate go run github.com/vektra/mockery/v2 --name=UserService --case=underscore

// UserService defines the interface for user business logic.
// This abstraction allows for easy testing of handlers.
type UserService interface {
	ListUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error)
}

// UserHandler handles HTTP requests for user endpoints.
type UserHandler struct {
	service UserService
}

// NewUserHandler creates a new UserHandler with the given service.
func NewUserHandler(service UserService) *UserHandler {
	if service == nil {
		panic("service cannot be nil")
	}
	return &UserHandler{service: service}
}

// ServeHTTP routes requests to the appropriate handler method.
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listUsers handles GET /users requests.
func (h *UserHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, users)
}

// createUser handles POST /users requests.
func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req domain.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp, err := h.service.CreateUser(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// respondJSON writes a JSON response with the given status code.
func (h *UserHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but cannot change headers after WriteHeader
		// In production, use a proper logger
		_ = err
	}
}

// handleError maps domain errors to HTTP status codes.
func (h *UserHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, domain.ErrDatabase):
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
