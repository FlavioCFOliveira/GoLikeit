// Package handler contains HTTP handlers for the application.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/example/project/models"
	"github.com/example/project/service"
)

// UserHandler handles HTTP requests for user operations.
type UserHandler struct {
	service *service.UserService
	logger  *slog.Logger
}

// NewUserHandler creates a new user handler with dependency injection.
func NewUserHandler(service *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// ServeHTTP routes requests to the appropriate handler method.
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetUsers(w, r)
	case http.MethodPost:
		h.handleCreateUser(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGetUsers handles GET /users requests.
func (h *UserHandler) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to retrieve users")
		return
	}

	h.writeJSON(w, http.StatusOK, users)
}

// handleCreateUser handles POST /users requests.
func (h *UserHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	defer r.Body.Close()

	user, err := h.service.CreateUser(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]int64{"id": user.ID})
}

// handleServiceError converts service errors to appropriate HTTP responses.
func (h *UserHandler) handleServiceError(w http.ResponseWriter, err error) {
	// Check for validation errors
	if errors.Is(err, models.ErrUserNameRequired) || errors.Is(err, models.ErrUserNameTooLong) {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log internal errors but don't expose details
	h.logger.Error("service error", "error", err)
	h.writeError(w, http.StatusInternalServerError, "internal server error")
}

// writeJSON writes a JSON response with the given status code.
func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// writeError writes a JSON error response.
func (h *UserHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
