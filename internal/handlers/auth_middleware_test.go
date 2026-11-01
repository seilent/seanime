package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyRoutesNeedAuth(t *testing.T) {
	h := &Handler{}
	require.False(t, h.isPublicPath("/api/v1/proxy"))
	require.False(t, h.isPublicPath("/api/v1/image-proxy"))
	require.True(t, h.isPublicPath("/api/v1/users/login"))
}
