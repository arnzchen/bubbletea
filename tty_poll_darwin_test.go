package tea

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func countReaderPollDescriptors(t *testing.T) int {
	t.Helper()
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof required to count kqueue descriptors")
	}
	output, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(os.Getpid()), "-F", "t").Output()
	if err != nil {
		t.Fatalf("count descriptors: %v", err)
	}
	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		if line == "tKQUEUE" {
			count++
		}
	}
	return count
}
