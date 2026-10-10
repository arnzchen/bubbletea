//go:build !linux && !darwin

package tea

import "testing"

func countReaderPollDescriptors(t *testing.T) int {
	t.Helper()
	t.Skip("poll descriptor measurement requires Linux or macOS")
	return 0
}
