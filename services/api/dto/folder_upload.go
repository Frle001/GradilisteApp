package dto

// FolderUploadResult is the per-file outcome of a batch folder upload.
type FolderUploadResult struct {
	RelativePath string  `json:"relative_path"`
	Status       string  `json:"status"` // "uploaded" | "skipped" | "rejected" | "failed"
	DocumentID   *string `json:"document_id,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// FolderUploadSummary aggregates counts across all files in a batch.
type FolderUploadSummary struct {
	Total    int `json:"total"`
	Uploaded int `json:"uploaded"`
	Skipped  int `json:"skipped"`
	Rejected int `json:"rejected"`
	Failed   int `json:"failed"`
}

// FolderUploadResponse is the complete batch upload response body.
type FolderUploadResponse struct {
	Summary FolderUploadSummary  `json:"summary"`
	Results []FolderUploadResult `json:"results"`
}
