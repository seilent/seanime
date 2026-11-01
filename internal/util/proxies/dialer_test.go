package util

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestIsPublicIP(t *testing.T) {
	for _, addr := range []string{"127.0.0.1", "::1", "10.2.0.2", "192.168.200.3", "172.26.69.50", "100.64.1.1", "169.254.169.254", "0.0.0.0", "0.1.2.3", "fd00::1", "fe80::1", "::ffff:127.0.0.1", "::",
		"198.18.0.1", "240.0.0.1", "192.0.0.8", "64:ff9b::7f00:1", "2002:7f00:1::", "2001:0:4136:e378::1", "::7f00:1", "fec0::1"} {
		require.False(t, isPublicIP(net.ParseIP(addr)), addr)
	}
	addrs, err := net.InterfaceAddrs()
	require.NoError(t, err)
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			require.False(t, isPublicIP(n.IP), n.IP.String())
		}
	}
	for _, addr := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		require.True(t, isPublicIP(net.ParseIP(addr)), addr)
	}
	require.NoError(t, publicOnlyControl("tcp", "1.1.1.1:443", nil))
	require.ErrorIs(t, publicOnlyControl("tcp", "127.0.0.1:8080", nil), errBlockedDestination)
}

func TestProxiesRefuseLoopback(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("secret"))
	}))
	defer srv.Close()

	_, err := (&ImageProxy{}).GetImage(srv.URL, map[string]string{})
	require.ErrorIs(t, err, errBlockedDestination)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy?url="+url.QueryEscape(srv.URL), nil)
	rec := httptest.NewRecorder()
	err = VideoProxy(e.NewContext(req, rec))
	require.Error(t, err)
	require.NotContains(t, rec.Body.String(), "secret")

	require.EqualValues(t, 0, hits.Load())
}

func TestImageProxyServesOnlyImages(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page.png" {
			_, _ = w.Write(png)
			return
		}
		_, _ = w.Write([]byte("<html><script>alert(1)</script></html>"))
	}))
	defer srv.Close()

	saved := imageProxyClient
	imageProxyClient = srv.Client()
	defer func() { imageProxyClient = saved }()

	e := echo.New()
	get := func(target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/image-proxy?headers=%7B%7D&url="+url.QueryEscape(target), nil)
		rec := httptest.NewRecorder()
		require.NoError(t, (&ImageProxy{}).ProxyImage(e.NewContext(req, rec)))
		return rec
	}

	rec := get(srv.URL + "/page.png")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))

	rec = get(srv.URL + "/page.html")
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotContains(t, rec.Body.String(), "script")
}
