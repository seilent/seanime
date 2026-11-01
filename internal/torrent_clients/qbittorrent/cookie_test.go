package qbittorrent

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"seanime/internal/util"
	"strconv"
	"sync/atomic"
	"testing"
)

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	return NewClient(&NewClientOptions{
		Logger:   util.NewLogger(),
		Username: "admin",
		Password: "secret",
		Host:     u.Hostname(),
		Port:     port,
	})
}

func TestBypassSessionCookieReused(t *testing.T) {
	var sessions atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("SID")
		if err != nil {
			n := sessions.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: fmt.Sprintf("sid%d", n), Path: "/", HttpOnly: true})
		} else if c.Value != "sid1" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("v5.1.0"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	for i := 0; i < 5; i++ {
		v, err := client.Application.GetAppVersion()
		require.NoError(t, err)
		require.Equal(t, "v5.1.0", v)
	}
	require.EqualValues(t, 1, sessions.Load())
}

func TestLoginCookieUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			_ = r.ParseForm()
			if r.PostForm.Get("username") != "admin" || r.PostForm.Get("password") != "secret" {
				_, _ = w.Write([]byte("Fails."))
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "login", Path: "/", HttpOnly: true})
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if c, err := r.Cookie("SID"); err != nil || c.Value != "login" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("v5.1.0"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.Application.GetAppVersion()
	require.Error(t, err)
	require.NoError(t, client.Login())
	v, err := client.Application.GetAppVersion()
	require.NoError(t, err)
	require.Equal(t, "v5.1.0", v)
}
