package manager

import (
	"testing"
	"time"
)

func TestWatchServerOptions(t *testing.T) {
	opts := WatchServerOptions{
		Port:         8080,
		Interval:     5 * time.Minute,
		NotifyMethod: "system",
		Quiet:        false,
	}

	if opts.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", opts.Port)
	}

	if opts.Interval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", opts.Interval)
	}
}

func TestConflictInfo(t *testing.T) {
	conflict := ConflictInfo{
		Branch:        "feature/test",
		WorktreePath:  "/path/to/worktree",
		ConflictFiles: []string{"file1.go", "file2.go"},
		LastChecked:   time.Now(),
	}

	if conflict.Branch != "feature/test" {
		t.Errorf("Expected branch feature/test, got %s", conflict.Branch)
	}

	if len(conflict.ConflictFiles) != 2 {
		t.Errorf("Expected 2 conflict files, got %d", len(conflict.ConflictFiles))
	}
}

func TestHasNewConflicts(t *testing.T) {
	oldConflicts := []ConflictInfo{
		{
			Branch:        "feature/old",
			ConflictFiles: []string{"file1.go"},
		},
	}

	newConflicts := []ConflictInfo{
		{
			Branch:        "feature/old",
			ConflictFiles: []string{"file1.go"},
		},
		{
			Branch:        "feature/new",
			ConflictFiles: []string{"file2.go"},
		},
	}

	if !HasNewConflicts(oldConflicts, newConflicts) {
		t.Error("Expected new conflicts to be detected")
	}

	if HasNewConflicts(oldConflicts, oldConflicts) {
		t.Error("Expected no new conflicts when comparing same lists")
	}
}

func TestParseConflictFiles(t *testing.T) {
	output := `CONFLICT (content): Merge conflict in src/main.go
CONFLICT (content): Merge conflict in pkg/utils.go
Some other output
CONFLICT (rename): Merge conflict in old.txt`

	files := parseConflictFiles(output)

	if len(files) != 3 {
		t.Errorf("Expected 3 conflict files, got %d", len(files))
	}

	expected := []string{"src/main.go", "pkg/utils.go", "old.txt"}
	for i, file := range files {
		if file != expected[i] {
			t.Errorf("Expected file %s, got %s", expected[i], file)
		}
	}
}
