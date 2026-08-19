package openapi

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateToken(t *testing.T) {
	first, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("generated tokens must be unique")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("token is not URL-safe base64: %v", err)
	}
	if len(decoded) != 24 {
		t.Fatalf("token contains %d random bytes, want 24", len(decoded))
	}
}

func TestEndpointForRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "http://internal:3002/api/openApi/getConfig", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "panel.example.com")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	if endpoint := endpointForRequest(context); endpoint != "https://panel.example.com/openapi/v1" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
}

func TestFetchImageDataURLDetectsImageContent(t *testing.T) {
	imageBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")
		_, _ = response.Write(imageBytes)
	}))
	defer server.Close()

	dataURL, err := fetchImageDataURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("unexpected data URL: %s", dataURL)
	}
}
