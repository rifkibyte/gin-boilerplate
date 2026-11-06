package file

// UploadResponse is returned after a successful upload.
type UploadResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
