package secrets

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnvDestination(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		expected    bool
	}{
		{"valid env var", "MY_SECRET", true},
		{"valid env var with underscores", "MY_SECRET_KEY", true},
		{"valid env var with numbers", "SECRET_123", true},
		{"starts with underscore", "_SECRET", true},
		{"lowercase", "my_secret", false},
		{"starts with number", "123_SECRET", false},
		{"file path", "/home/dev/.kube/config", false},
		{"relative path", "~/config", false},
		{"contains slash", "MY/SECRET", false},
		{"contains dash", "MY-SECRET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{Destination: tt.destination}
			assert.Equal(t, tt.expected, s.IsEnvDestination())
		})
	}
}

func TestExpandedDestination(t *testing.T) {
	os.Setenv("TEST_VAR", "/test/path")
	defer os.Unsetenv("TEST_VAR")

	homeDir := os.Getenv("HOME")

	tests := []struct {
		name        string
		destination string
		expected    string
	}{
		{"env var name", "MY_SECRET", "MY_SECRET"},
		{"absolute path", "/etc/secrets/config", "/etc/secrets/config"},
		{"tilde expansion", "~/.kube/config", homeDir + "/.kube/config"},
		{"env var in path", "$TEST_VAR/file", "/test/path/file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{Destination: tt.destination}
			result, err := s.ExpandedDestination()
      
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateDestination(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		expectError bool
	}{
		{"empty destination", "", true},
		{"valid env var", "MY_SECRET", false},
		{"valid kube path", "/home/dev/.kube/config", false},
		{"valid ssh path", "/home/dev/.ssh/id_rsa", false},
		{"valid secrets path", "/etc/secrets/token", false},
		{"invalid path", "/tmp/secret", true},
		{"invalid path home", "/home/user/file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{Destination: tt.destination}
			err := s.ValidateDestination()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
