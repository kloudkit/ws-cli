package net

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDefaultGateway(t *testing.T) {
	ip, err := defaultGateway("Iface\tDestination\tGateway\neth0\t0001A8C0\t00000000\neth0\t00000000\t0101A8C0\n")
	assert.NilError(t, err)
	assert.Equal(t, ip, "192.168.1.1")

	_, err = defaultGateway("Iface\tDestination\tGateway\n")
	assert.ErrorContains(t, err, "default gateway not found")
}
