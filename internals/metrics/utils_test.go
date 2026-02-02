package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestReadUint64FromFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "uint64")
	err := os.WriteFile(path, []byte(" 12345 \n"), 0644)
	assert.NilError(t, err)

	val, err := readUint64FromFile(path)
	assert.NilError(t, err)
	assert.Equal(t, val, uint64(12345))

	_, err = readUint64FromFile(filepath.Join(tempDir, "missing"))
	assert.Assert(t, err != nil)
}

func TestParseKVStats(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "kvstats")
	content := "key1 100\nkey2 200\nkey3 not_int\nkey4\n"
	err := os.WriteFile(path, []byte(content), 0644)
	assert.NilError(t, err)

	stats, err := parseKVStats(path)
	assert.NilError(t, err)
	assert.Equal(t, stats["key1"], uint64(100))
	assert.Equal(t, stats["key2"], uint64(200))
	_, ok := stats["key3"]
	assert.Assert(t, !ok)
}

func TestProcessFileLines(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "lines")
	content := "line1\nline2\n"
	err := os.WriteFile(path, []byte(content), 0644)
	assert.NilError(t, err)

	var lines []string
	err = processFileLines(path, func(line string) {
		lines = append(lines, line)
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, lines, []string{"line1", "line2"})
}
