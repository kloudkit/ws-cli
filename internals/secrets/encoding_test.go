package secrets

import (
	"encoding/base64"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncodeWithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "simple string",
			input:    []byte("hello world"),
			expected: "base64:aGVsbG8gd29ybGQ=",
		},
		{
			name:     "empty string",
			input:    []byte(""),
			expected: "base64:",
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: "base64:AQIDBA==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeWithPrefix(tt.input)
			assert.DeepEqual(t, tt.expected, result)
		})
	}
}

func TestDecodeWithPrefix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []byte
		shouldErr bool
	}{
		{
			name:     "with base64 prefix",
			input:    "base64:aGVsbG8gd29ybGQ=",
			expected: []byte("hello world"),
		},
		{
			name:     "without prefix - returns as-is",
			input:    "plain text",
			expected: []byte("plain text"),
		},
		{
			name:     "empty with prefix",
			input:    "base64:",
			expected: []byte(""),
		},
		{
			name:     "binary data with prefix",
			input:    "base64:AQIDBA==",
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:      "invalid base64",
			input:     "base64:invalid!!!",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeWithPrefix(tt.input)

			if tt.shouldErr {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
				assert.DeepEqual(t, tt.expected, result)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := []byte("test data with special chars: 你好世界 🚀")
	encoded := EncodeWithPrefix(original)
	decoded, err := DecodeWithPrefix(encoded)

	assert.NilError(t, err)
	assert.DeepEqual(t, original, decoded)
}

func TestDecodeRawEncrypted(t *testing.T) {
	rawEncrypted := "some-encrypted-string-without-prefix"
	decoded, err := DecodeWithPrefix(rawEncrypted)

	assert.NilError(t, err)
	assert.DeepEqual(t, []byte(rawEncrypted), decoded)
}

func TestEncodeMatchesManualEncoding(t *testing.T) {
	data := []byte("test")
	encoded := EncodeWithPrefix(data)
	manualEncoded := "base64:" + base64.StdEncoding.EncodeToString(data)

	assert.Equal(t, manualEncoded, encoded)
}
