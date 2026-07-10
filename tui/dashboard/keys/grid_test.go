package keys

import "testing"

func TestGetFileGridPositions(t *testing.T) {
	files := []string{"01_domain_model_use_cases.md", "file1.md", "file2.md", "file3.md"}
	srcIdx, grid := GetFileGridPositions(files, "01_domain_model_use_cases.md")
	if srcIdx != 0 {
		t.Errorf("expected source index 0, got %d", srcIdx)
	}
	if len(grid) != 2 {
		t.Errorf("expected 2 rows, got %d", len(grid))
	}
}

func TestNavigate(t *testing.T) {
	files := []string{"01_domain_model_use_cases.md", "file1.md", "file2.md", "file3.md"}
	
	// Test NavigateDown from source
	next := NavigateDown(0, files, "01_domain_model_use_cases.md")
	if next != 1 {
		t.Errorf("expected navigate down from source to go to index 1, got %d", next)
	}

	// Test NavigateLeft/Right
	next = NavigateRight(1, files, "01_domain_model_use_cases.md")
	if next != 3 {
		t.Errorf("expected navigate right to go to index 3, got %d", next)
	}
	
	next = NavigateLeft(3, files, "01_domain_model_use_cases.md")
	if next != 1 {
		t.Errorf("expected navigate left to go back to 1, got %d", next)
	}
}
