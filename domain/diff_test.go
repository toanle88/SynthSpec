package domain

import (
	"encoding/json"
	"testing"
)

func TestFileDiff_Serialization(t *testing.T) {
	fd := FileDiff{
		FileName:   "test.md",
		OldContent: "hello",
		NewContent: "world",
		DiffText:   "-hello\n+world",
	}

	data, err := json.Marshal(fd)
	if err != nil {
		t.Fatalf("failed to marshal FileDiff: %v", err)
	}

	var fd2 FileDiff
	if err := json.Unmarshal(data, &fd2); err != nil {
		t.Fatalf("failed to unmarshal FileDiff: %v", err)
	}

	if fd2.FileName != fd.FileName || fd2.DiffText != fd.DiffText {
		t.Errorf("mismatch: %+v vs %+v", fd, fd2)
	}
}
