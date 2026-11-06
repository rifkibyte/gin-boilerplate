package file

import "time"

// File is the domain entity for an uploaded file.
type File struct {
	ID          string
	UserID      string
	Filename    string
	ContentType string
	Size        int64
	Path        string
	CreatedAt   time.Time
}
