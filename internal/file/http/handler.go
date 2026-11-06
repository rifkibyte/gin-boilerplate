package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rifkibyte/gin-boilerplate/internal/file"
	"github.com/rifkibyte/gin-boilerplate/pkg/response"
)

type Handler struct {
	svc *file.Service
}

func NewHandler(svc *file.Service) *Handler {
	return &Handler{svc: svc}
}

const maxUploadSize = 10 * 1024 * 1024

// Upload godoc
// @Summary Upload a file
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 201 {object} file.UploadResponse
// @Failure 400 {object} response.ErrorEnvelope
// @Failure 413 {object} response.ErrorEnvelope
// @Router /files [post]
func (h *Handler) Upload(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+1024)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "missing file")
		return
	}
	if fileHeader.Size > maxUploadSize {
		response.Error(c, http.StatusRequestEntityTooLarge, "file too large")
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "cannot open file")
		return
	}
	defer func() { _ = src.Close() }()

	cmd := file.UploadCommand{
		UserID:      userID,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
		Content:     src,
	}

	f, err := h.svc.Upload(c.Request.Context(), cmd)
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	response.Created(c, file.UploadResponse{
		ID:          f.ID,
		Filename:    f.Filename,
		ContentType: f.ContentType,
		Size:        f.Size,
	})
}

// List godoc
// @Summary List files for current user
// @Tags files
// @Produce json
// @Success 200 {array} file.UploadResponse
// @Failure 401 {object} response.ErrorEnvelope
// @Router /files [get]
func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	files, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.FromError(c, toAppError(err))
		return
	}

	resp := make([]file.UploadResponse, 0, len(files))
	for _, f := range files {
		resp = append(resp, file.UploadResponse{
			ID:          f.ID,
			Filename:    f.Filename,
			ContentType: f.ContentType,
			Size:        f.Size,
		})
	}

	response.OK(c, resp)
}
