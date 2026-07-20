package dto

import "time"

type ProjectDocumentItem struct {
	ID              string    `json:"id"`
	OriginalName    string    `json:"original_name"`
	ContentType     string    `json:"content_type"`
	FileSize        int64     `json:"file_size"`
	UploadedByEmail string    `json:"uploaded_by_email"`
	CreatedAt       time.Time `json:"created_at"`
	FolderID        *string   `json:"folder_id,omitempty"`
}

type ProjectFolderItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
