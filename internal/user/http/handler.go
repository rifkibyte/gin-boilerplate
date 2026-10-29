package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/internal/user"
	"github.com/rifkibyte/gin-boilerplate/pkg/response"
)

// Handler is the HTTP adapter for the user feature.
type Handler struct {
	svc *user.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *user.Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary Create user
// @Tags users
// @Accept json
// @Produce json
// @Param request body user.CreateUserRequest true "Create payload"
// @Success 201 {object} user.UserResponse
// @Router /users [post]
// Create handles POST /users.
func (h *Handler) Create(c *gin.Context) {
	var req user.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	cmd := user.CreateUserCommand(req)

	u, err := h.svc.Create(c.Request.Context(), cmd)
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.Created(c, toUserResponse(u))
}

// List godoc
// @Summary List users
// @Tags users
// @Produce json
// @Success 200 {array} user.UserResponse
// @Router /users [get]
// List handles GET /users.
func (h *Handler) List(c *gin.Context) {
	users, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	resp := make([]user.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toUserResponse(u))
	}

	response.OK(c, resp)
}

// toUserResponse maps a domain entity to a response DTO.
func toUserResponse(u *user.User) user.UserResponse {
	return user.UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
