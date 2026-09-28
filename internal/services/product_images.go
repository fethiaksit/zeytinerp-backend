package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

const productImagePrefix = "/api/uploads/products/"
const maxProductImageBytes = 5 * 1024 * 1024

var productImageName = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|gif|webp)$`)

func ProductImageDir() string {
	if dir := strings.TrimSpace(os.Getenv("PRODUCT_IMAGE_DIR")); dir != "" {
		return dir
	}
	return "uploads/products"
}

func ProductImagePath(name string) (string, error) {
	if !productImageName.MatchString(name) {
		return "", errors.New("Ürün görsel yolu geçersiz.")
	}
	return filepath.Join(ProductImageDir(), name), nil
}

func validateImageURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return errors.New("Ürün görseli için geçerli bir http/https adresi giriniz.")
	}
	if port := u.Port(); port != "" && port != "80" && port != "443" {
		return errors.New("Ürün görsel adresinin portu geçersiz.")
	}
	return nil
}

func publicImageIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !net.ParseIP("100.64.0.0").Equal(ip.Mask(net.CIDRMask(10, 32)))
}

// Resolve and dial the same validated IP to prevent redirects or DNS rebinding
// from making the server fetch its own network or cloud metadata.
func dialProductImage(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("empty DNS response")
	}
	for _, ip := range ips {
		if !publicImageIP(ip.IP) {
			return nil, errors.New("private image address")
		}
	}
	var last error
	for _, ip := range ips {
		conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

var productImageClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: &http.Transport{DialContext: dialProductImage, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, MaxIdleConns: 10, IdleConnTimeout: 30 * time.Second},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return validateImageURL(req.URL.String())
	},
}

func StoreProductImage(ctx context.Context, source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil
	}
	if strings.HasPrefix(source, productImagePrefix) {
		path, err := ProductImagePath(strings.TrimPrefix(source, productImagePrefix))
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return "", errors.New("Kayıtlı ürün görseli bulunamadı. Görsel adresini yeniden giriniz.")
		}
		return source, nil
	}
	return downloadProductImage(ctx, source, productImageClient)
}

func downloadProductImage(ctx context.Context, source string, client *http.Client) (string, error) {
	if err := validateImageURL(source); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", errors.New("Ürün görsel adresi geçersiz.")
	}
	req.Header.Set("User-Agent", "ZeytinERP/1.0 ProductImageDownloader")
	req.Header.Set("Accept", "image/jpeg,image/png,image/webp,image/gif")
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("Ürün görseli indirilemedi. Adresi ve sunucunun internet bağlantısını kontrol edin.")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ürün görseli indirilemedi (HTTP %d). Başka bir doğrudan görsel adresi deneyin.", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxProductImageBytes+1))
	if err != nil {
		return "", errors.New("Ürün görseli okunamadı. Tekrar deneyin.")
	}
	if len(data) > maxProductImageBytes {
		return "", errors.New("Ürün görseli 5 MB sınırını aşıyor.")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 25_000_000 {
		return "", errors.New("Ürün görseli geçersiz veya çok büyük. JPEG, PNG, GIF veya WebP görsel adresi giriniz.")
	}
	extensions := map[string]string{"jpeg": "jpg", "png": "png", "gif": "gif", "webp": "webp"}
	ext, ok := extensions[format]
	if !ok {
		return "", errors.New("Ürün görsel biçimi desteklenmiyor.")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return "", errors.New("Ürün görsel dosyası bozuk veya eksik.")
	}
	name := fmt.Sprintf("%x.%s", sha256.Sum256(data), ext)
	dir := ProductImageDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", errors.New("Ürün görsel klasörü oluşturulamadı. Sunucu yazma izinlerini kontrol edin.")
	}
	file, err := os.CreateTemp(dir, ".image-*")
	if err != nil {
		return "", errors.New("Ürün görseli sunucuya kaydedilemedi. Yazma izinlerini kontrol edin.")
	}
	temp := file.Name()
	defer os.Remove(temp)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return "", errors.New("Ürün görseli sunucuya yazılamadı.")
	}
	if err := os.Rename(temp, filepath.Join(dir, name)); err != nil {
		return "", errors.New("Ürün görseli sunucuya kaydedilemedi.")
	}
	return productImagePrefix + name, nil
}
