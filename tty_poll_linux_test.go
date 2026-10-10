package tea

import (
	"os"
	"path/filepath"
	"testing"
)

func countReaderPollDescriptors(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err == nil && target == "anon_inode:[eventpoll]" {
			count++
		}
	}
	return count
}
