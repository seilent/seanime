package mediastream

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestServeFileInDirStaysInside(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "videofiles", "hash", "subs")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0.ass"), []byte("sub"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "seanime.db"), []byte("secret"), 0644))

	e := echo.New()
	e.GET("/subs/*", func(c echo.Context) error {
		return serveFileInDir(c, dir, c.Param("*"))
	})

	get := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}

	rec := get("/subs/0.ass")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "sub", rec.Body.String())

	for _, target := range []string{"/subs/../../../seanime.db", "/subs/..%2F..%2F..%2Fseanime.db", "/subs/", "/subs/.."} {
		rec = get(target)
		require.Equal(t, http.StatusNotFound, rec.Code, target)
		require.NotContains(t, rec.Body.String(), "secret", target)
	}
}
