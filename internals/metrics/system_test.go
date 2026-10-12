package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetFileDescriptorStatsCountsEveryProcess(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { procRoot = "/proc" })
	procRoot = root

	fds := map[string]int{"1": 3, "42": 5, "self": 2}
	for pid, count := range fds {
		dir := filepath.Join(root, pid, "fd")
		assert.NilError(t, os.MkdirAll(dir, 0o755))

		for i := range count {
			assert.NilError(t, os.WriteFile(filepath.Join(dir, string(rune('a'+i))), nil, 0o644))
		}
	}

	assert.NilError(t, os.MkdirAll(filepath.Join(root, "sys"), 0o755))
	assert.NilError(t, os.WriteFile(
		filepath.Join(root, "self", "limits"),
		[]byte("Limit                     Soft Limit           Hard Limit           Units\nMax open files            1048576              1048576              files\n"),
		0o644,
	))

	stats, err := GetFileDescriptorStats()
	assert.NilError(t, err)

	assert.Equal(t, stats.Open, uint64(8))
	assert.Equal(t, stats.Limit, uint64(1048576))
}

func TestGetFileDescriptorStatsWithoutReadableProcesses(t *testing.T) {
	t.Cleanup(func() { procRoot = "/proc" })
	procRoot = t.TempDir()

	_, err := GetFileDescriptorStats()
	assert.ErrorContains(t, err, "no readable file descriptor tables")
}
