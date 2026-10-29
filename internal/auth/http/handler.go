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