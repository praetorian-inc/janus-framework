package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScannableAsset_HasPort(t *testing.T) {
	sa := NewScannableAsset("192.168.1.1")
	sa.Ports = []Port{{Transport: "tcp", Port: "80"}, {Transport: "tcp", Port: "443"}}
	assert.True(t, sa.HasPort("tcp", "80"))
	assert.False(t, sa.HasPort("tcp", "22"))
	assert.False(t, sa.HasPort("udp", "80"))
}

func TestScannableAsset_AddPort(t *testing.T) {
	sa := NewScannableAsset("192.168.1.1")
	assert.False(t, sa.HasPort("tcp", "80"))
	sa.AddPort("tcp", "80")
	assert.True(t, sa.HasPort("tcp", "80"))
}
