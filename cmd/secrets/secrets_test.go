package secrets

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGenerate(t *testing.T) {
	buffer := new(bytes.Buffer)
	cmd := SecretsCmd
	cmd.SetOut(buffer)
	cmd.SetArgs([]string{"generate", "--length", "16", "--raw"})

	err := cmd.Execute()
	assert.NilError(t, err)

	output := buffer.String()
	assert.Equal(t, len(strings.TrimSpace(output)), 24)
}
