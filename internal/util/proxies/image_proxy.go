package util

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"seanime/internal/util"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type ImageProxy struct{}

var imageProxyClient = &http.Client{
	Transport: &http.Transport{DialContext: publicDialer.DialContext},
	Timeout:   60 * time.Second,
}

func (ip *ImageProxy) GetImage(url string, headers map[string]string) ([]byte, error) {
	body, _, err := fetchImage(url, headers)
	return body, err
}

func fetchImage(url string, headers map[string]string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := imageProxyClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return body, resp.Header.Get("Content-Type"), nil
}

func imageContentType(body []byte, upstream string) (string, bool) {
	if sniffed := http.DetectContentType(body); strings.HasPrefix(sniffed, "image/") {
		return sniffed, true
	}
	mediaType, _, err := mime.ParseMediaType(upstream)
	if err != nil || !strings.HasPrefix(mediaType, "image/") || mediaType == "image/svg+xml" {
		return "", false
	}
	return mediaType, true
}

func (ip *ImageProxy) setHeaders(c echo.Context) {
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
}

func (ip *ImageProxy) ProxyImage(c echo.Context) (err error) {
	defer util.HandlePanicInModuleWithError("util/ImageProxy", &err)

	url := c.QueryParam("url")
	headersJSON := c.QueryParam("headers")

	if url == "" || headersJSON == "" {
		return c.String(echo.ErrBadRequest.Code, "No URL provided")
	}

	headers := make(map[string]string)
	if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
		return c.String(echo.ErrBadRequest.Code, "Error parsing headers JSON")
	}

	ip.setHeaders(c)
	imageBuffer, upstreamType, err := fetchImage(url, headers)
	if err != nil {
		return c.String(echo.ErrInternalServerError.Code, "Error fetching image")
	}

	contentType, ok := imageContentType(imageBuffer, upstreamType)
	if !ok {
		return c.String(http.StatusBadGateway, "Not an image")
	}
	c.Response().Header().Set("Cache-Control", "private, max-age=31536000")
	return c.Blob(http.StatusOK, contentType, imageBuffer)
}
