package handlers

import "testing"

func TestValidateImageSourceRejectsInternalTargets(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/photo.jpg", "http://localhost/photo.jpg",
		"http://169.254.169.254/latest/meta-data/", "http://10.0.0.1/a.png",
		"http://192.168.1.1/a.png", "http://[::1]/a.png", "http://[::ffff:127.0.0.1]/a.png",
		"file:///etc/passwd", "https://user:pass@example.com/a.jpg",
	} {
		if err := validateImageSource(raw); err == nil { t.Errorf("accepted private or invalid source %q", raw) }
	}
	if err := validateImageSource("https://images.example.com/product.png"); err != nil { t.Fatalf("valid public URL rejected: %v", err) }
}

func TestValidateDownloadedImageRejectsHTMLAndOversize(t *testing.T) {
	if _, err := validateDownloadedImage([]byte("<html>not an image</html>")); err == nil { t.Fatal("HTML accepted as an image") }
	if _, err := validateDownloadedImage(make([]byte, maxProductImageSize+1)); err == nil { t.Fatal("oversize image accepted") }
	if _, err := validateDownloadedImage([]byte("RIFFxxxxWEBPVP8Xxxxxxxxxxxxxxxxxxx")); err == nil { t.Fatal("malformed WebP accepted") }
}
