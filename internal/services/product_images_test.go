package services

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadProductImagePersistsValidatedBytes(t *testing.T) {
	t.Setenv("PRODUCT_IMAGE_DIR", t.TempDir())
	var data bytes.Buffer
	png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data.Bytes())), Header: http.Header{}}, nil
	})}
	path, err := downloadProductImage(context.Background(), "https://example.com/photo.jpg", client)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, "/api/uploads/products/") || !strings.HasSuffix(path, ".png") {
		t.Fatal(path)
	}
	got, err := os.ReadFile(filepath.Join(ProductImageDir(), filepath.Base(path)))
	if err != nil || !bytes.Equal(got, data.Bytes()) {
		t.Fatalf("saved bytes mismatch: %v", err)
	}
	cached, err := StoreProductImage(context.Background(), path)
	if err != nil || cached != path {
		t.Fatalf("cache reuse: %s %v", cached, err)
	}
}

func TestDownloadProductImageRejectsNonImagesAndOversize(t *testing.T) {
	for _, body := range []string{"<html>not an image</html>", strings.Repeat("x", 5*1024*1024+1)} {
		client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
		})}
		if _, err := downloadProductImage(context.Background(), "https://example.com/photo.jpg", client); err == nil {
			t.Fatal("invalid image accepted")
		}
	}
}

func TestStoreProductImageRejectsUnsafeSources(t *testing.T) {
	for _, source := range []string{"not-a-url", "file:///etc/passwd", "http://127.0.0.1/a.jpg", "http://169.254.169.254/a", "http://[::1]/a", "/api/uploads/products/../../secret", "https://user:pass@example.com/a"} {
		if _, err := StoreProductImage(context.Background(), source); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
}
