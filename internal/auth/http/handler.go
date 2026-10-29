package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/internal/auth"
	"github.com/rifkibyte/gin-boilerplate/internal/user"
	"github.com/rifkibyte/gin-boilerplate/pkg/response"
)

type Handler struct {
	svc *auth.Service
}

func NewHandler(svc *auth.Service) *Handler {
	return &Handler{svc: svc}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Register payload"
// @Success 201 {object} user.UserResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	u, err := h.svc.Register(c.Request.Context(), auth.RegisterCommand(req))
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.Created(c, user.UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Login godoc
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login payload"
// @Success 200 {object} auth.TokenResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), auth.LoginCommand(req))
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.OK(c, auth.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

// Refresh godoc
// @Summary Refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshRequest true "Refresh payload"
// @Success 200 {object} auth.TokenResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req auth.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.OK(c, auth.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

// Logout godoc
// @Summary Logout user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshRequest true "Logout payload"
// @Success 204
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req auth.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.NoContent(c)
}