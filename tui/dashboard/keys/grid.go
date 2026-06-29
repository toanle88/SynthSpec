package keys

// GridHelper provides pure utility functions for file grid navigation.
// These are stateless computations extracted from DashboardModel methods,
// placed in a subpackage to support the directory structure goal.

// GetFileGridPositions computes the 2D grid layout of file indices.
// Returns the source file index and a 2D grid of downstream file indices.
func GetFileGridPositions(genFiles []string, domainModelFilename string) (int, [][]int) {
	sourceIdx := -1
	var downstream []int
	for idx, file := range genFiles {
		if file == domainModelFilename {
			sourceIdx = idx
		} else {
			downstream = append(downstream, idx)
		}
	}

	if len(downstream) == 0 {
		return sourceIdx, nil
	}

	half := (len(downstream) + 1) / 2
	var grid [][]int
	for i := 0; i < half; i++ {
		row := []int{downstream[i]}
		if half+i < len(downstream) {
			row = append(row, downstream[half+i])
		}
		grid = append(grid, row)
	}
	return sourceIdx, grid
}

// GetGridPos determines whether the selected index is the source file or is in the downstream grid.
func GetGridPos(selected int, sourceIdx int, grid [][]int) (bool, int, int) {
	if selected == sourceIdx {
		return true, 0, 0
	}
	for r, rowFiles := range grid {
		for c, idx := range rowFiles {
			if idx == selected {
				return false, r, c
			}
		}
	}
	return true, 0, 0
}

// NavigateUp computes the new selected file index when navigating up.
func NavigateUp(selectedFileIdx int, genFiles []string, domainModelFilename string) int {
	if len(genFiles) == 0 {
		return 0
	}
	sourceIdx, grid := GetFileGridPositions(genFiles, domainModelFilename)
	if len(grid) == 0 {
		return 0
	}
	isSource, row, col := GetGridPos(selectedFileIdx, sourceIdx, grid)
	if isSource {
		return grid[len(grid)-1][0]
	}
	if row > 0 {
		if col < len(grid[row-1]) {
			return grid[row-1][col]
		}
		return grid[row-1][0]
	}
	if sourceIdx != -1 {
		return sourceIdx
	}
	return grid[len(grid)-1][0]
}

// NavigateDown computes the new selected file index when navigating down.
func NavigateDown(selectedFileIdx int, genFiles []string, domainModelFilename string) int {
	if len(genFiles) == 0 {
		return 0
	}
	sourceIdx, grid := GetFileGridPositions(genFiles, domainModelFilename)
	if len(grid) == 0 {
		return 0
	}
	isSource, row, col := GetGridPos(selectedFileIdx, sourceIdx, grid)
	if isSource {
		return grid[0][0]
	}
	if row < len(grid)-1 {
		if col < len(grid[row+1]) {
			return grid[row+1][col]
		}
		return grid[row+1][0]
	}
	if sourceIdx != -1 {
		return sourceIdx
	}
	return grid[0][col]
}

// NavigateLeft computes the new selected file index when navigating left.
func NavigateLeft(selectedFileIdx int, genFiles []string, domainModelFilename string) int {
	if len(genFiles) == 0 {
		return 0
	}
	sourceIdx, grid := GetFileGridPositions(genFiles, domainModelFilename)
	if len(grid) == 0 {
		return 0
	}
	isSource, row, col := GetGridPos(selectedFileIdx, sourceIdx, grid)
	if isSource {
		return selectedFileIdx
	}
	if col > 0 {
		return grid[row][col-1]
	}
	return grid[row][len(grid[row])-1]
}

// NavigateRight computes the new selected file index when navigating right.
func NavigateRight(selectedFileIdx int, genFiles []string, domainModelFilename string) int {
	if len(genFiles) == 0 {
		return 0
	}
	sourceIdx, grid := GetFileGridPositions(genFiles, domainModelFilename)
	if len(grid) == 0 {
		return 0
	}
	isSource, row, col := GetGridPos(selectedFileIdx, sourceIdx, grid)
	if isSource {
		return selectedFileIdx
	}
	if col < len(grid[row])-1 {
		return grid[row][col+1]
	}
	return grid[row][0]
}
