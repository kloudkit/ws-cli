package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func write(t *testing.T, path, content string) string {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestReadUint64FromFile(t *testing.T) {
	dir := t.TempDir()

	val, err := readUint64FromFile(write(t, filepath.Join(dir, "uint64"), " 12345 \n"))
	assert.NilError(t, err)
	assert.Equal(t, val, uint64(12345))

	_, err = readUint64FromFile(filepath.Join(dir, "missing"))
	assert.Assert(t, err != nil)
	assert.Equal(t, readUint(filepath.Join(dir, "missing")), uint64(0))
}

func TestParseKVStats(t *testing.T) {
	stats, err := parseKVStats(write(t, filepath.Join(t.TempDir(), "kv"), "key1 100\nkey2 200\n\nkey3 not_int\nkey4\n"))
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]uint64{"key1": 100, "key2": 200})
}

func TestScanFieldsSkipsHeaderAndBlankLines(t *testing.T) {
	var lines [][]string
	err := scanFields(write(t, filepath.Join(t.TempDir(), "lines"), "header\na  b\n\nc\n"), 1, func(fields []string) {
		lines = append(lines, fields)
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, lines, [][]string{{"a", "b"}, {"c"}})
}

func TestParseGPUStatsReadsFirstGPU(t *testing.T) {
	stats := parseGPUStats("45, 1024, 8192, 61, 120.5\n10, 1, 2, 3, 4\n")

	assert.DeepEqual(t, stats, &GPUStats{
		Available:          true,
		UtilizationRatio:   0.45,
		MemoryUsedBytes:    1024 << 20,
		MemoryTotalBytes:   8192 << 20,
		TemperatureCelsius: 61,
		PowerWatts:         120.5,
	})
	assert.Assert(t, !parseGPUStats("garbage").Available)
}

func TestProcStatsFromProcRoot(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { procRoot = "/proc" })
	procRoot = root

	for pid, count := range map[string]int{"1": 3, "42": 5, "self": 2} {
		for i := range count {
			write(t, filepath.Join(root, pid, "fd", string(rune('a'+i))), "")
		}
	}

	write(t, filepath.Join(root, "self", "limits"),
		"Limit                     Soft Limit           Hard Limit           Units\nMax open files            1024                 1048576              files\n")
	write(t, filepath.Join(root, "self", "net", "dev"),
		"Inter-|   Receive\n face |bytes\n  eth0:123456789 10 1 0 0 0 0 0 987 20 2 0 0 0 0 0\n    lo: 5 1 0 0 0 0 0 0 5 1 0 0 0 0 0 0\n")
	write(t, filepath.Join(root, "self", "net", "tcp"),
		"  sl local rem st\n   0: 0100007F:0035 00000000:0000 0A\n   1: 0100007F:0036 0100007F:0035 01\n")
	write(t, filepath.Join(root, "self", "net", "udp6"), "  sl local\n   0: x\n   1: y\n")

	fds, err := GetFileDescriptorStats()
	assert.NilError(t, err)
	assert.DeepEqual(t, fds, &FileDescriptorStats{Open: 8, Limit: 1024})

	network, err := GetNetworkStats()
	assert.NilError(t, err)
	assert.DeepEqual(t, network, &NetworkStats{
		ReceiveBytesTotal: 123456794, ReceivePacketsTotal: 11, ReceiveErrorsTotal: 1,
		TransmitBytesTotal: 992, TransmitPacketsTotal: 21, TransmitErrorsTotal: 2,
	})

	sockets, err := GetSocketStats()
	assert.NilError(t, err)
	assert.DeepEqual(t, sockets, &SocketStats{TCPEstablished: 1, TCPListen: 1, UDP: 2})
}

func TestGetFileDescriptorStatsWithoutReadableProcesses(t *testing.T) {
	t.Cleanup(func() { procRoot = "/proc" })
	procRoot = t.TempDir()

	_, err := GetFileDescriptorStats()
	assert.ErrorContains(t, err, "no readable file descriptor tables")
}

func TestBuildRegistryRejectsOnlyInvalidCollectors(t *testing.T) {
	_, err := BuildRegistry([]string{"bogus"})
	assert.ErrorContains(t, err, "no valid collectors in [bogus]")
}
