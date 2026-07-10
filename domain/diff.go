package domain

// FileDiff represents changes in a specific file.
type FileDiff struct {
	FileName   string `json:"file_name"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	DiffText   string `json:"diff_text"`
}
