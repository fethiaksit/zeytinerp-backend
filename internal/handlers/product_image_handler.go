package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"market-erp-backend/internal/models"
)

const maxProductImageSize = 5 << 20
const productImageDir = "uploads/products"

var imageFileName = regexp.MustCompile(`^product-[0-9]+-[0-9a-f]{32}\.(jpg|png|webp)$`)
var blockedImageNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"), netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2001::/32"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("64:ff9b::/96"),
}

func publicImageIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok { return false }
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() { return false }
	for _, block := range blockedImageNetworks { if block.Contains(addr) { return false } }
	return true
}

func validateImageSource(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return errors.New("Geçersiz görsel URL'si.")
	}
	if u.Port() != "" && !((u.Scheme == "http" && u.Port() == "80") || (u.Scheme == "https" && u.Port() == "443")) {
		return errors.New("Geçersiz görsel URL'si.")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if ip := net.ParseIP(host); ip != nil {
		if !publicImageIP(ip) { return errors.New("Geçersiz görsel URL'si.") }
		return nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || !strings.Contains(host, ".") {
		return errors.New("Geçersiz görsel URL'si.")
	}
	return nil
}

// Dial the validated public address directly. DNS cannot switch the connection
// to a private address after validation; this check also covers redirects.
func imageDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil { return nil, err }
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 { return nil, errors.New("Görsel indirilemedi.") }
	for _, addr := range addresses {
		if !publicImageIP(addr.IP) { return nil, errors.New("Geçersiz görsel URL'si.") }
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
}

func validateDownloadedImage(data []byte) (string, error) {
	if len(data) == 0 || len(data) > maxProductImageSize { return "", errors.New("Görsel boyutu çok büyük.") }
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png":
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 10000 || config.Height > 10000 { return "", errors.New("Bu URL geçerli bir görsel içermiyor.") }
		if format == "jpeg" { return ".jpg", nil }
		return ".png", nil
	case "image/webp":
		if !validWebPHeader(data) { return "", errors.New("Bu URL geçerli bir görsel içermiyor.") }
		return ".webp", nil
	default:
		return "", errors.New("Bu URL geçerli bir görsel içermiyor.")
	}
}

func validWebPHeader(data []byte) bool {
	if len(data) < 30 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || uint64(binary.LittleEndian.Uint32(data[4:8])) != uint64(len(data)-8) { return false }
	chunkSize := uint64(binary.LittleEndian.Uint32(data[16:20]))
	if chunkSize+20 > uint64(len(data)) { return false }
	var width, height int
	switch string(data[12:16]) {
	case "VP8X":
		if chunkSize < 10 { return false }
		width = 1 + int(data[24]) + int(data[25])<<8 + int(data[26])<<16
		height = 1 + int(data[27]) + int(data[28])<<8 + int(data[29])<<16
	case "VP8L":
		if chunkSize < 5 || data[20] != 0x2f { return false }
		width = 1 + int(data[21]) + int(data[22]&0x3f)<<8
		height = 1 + int(data[22]>>6) + int(data[23])<<2 + int(data[24]&0x0f)<<10
	case "VP8 ":
		if chunkSize < 10 || data[23] != 0x9d || data[24] != 0x01 || data[25] != 0x2a { return false }
		width = int(binary.LittleEndian.Uint16(data[26:28]) & 0x3fff)
		height = int(binary.LittleEndian.Uint16(data[28:30]) & 0x3fff)
	default:
		return false
	}
	return width > 0 && height > 0 && width <= 10000 && height <= 10000
}

func downloadProductImage(raw string) ([]byte, string, error) {
	if err := validateImageSource(raw); err != nil { return nil, "", err }
	transport := &http.Transport{Proxy: nil, DialContext: imageDial, TLSHandshakeTimeout: 5 * time.Second, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 { return errors.New("Görsel indirilemedi.") }
		return validateImageSource(req.URL.String())
	}}
	request, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil { return nil, "", errors.New("Geçersiz görsel URL'si.") }
	response, err := client.Do(request)
	if err != nil { return nil, "", errors.New("Görsel indirilemedi.") }
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK { return nil, "", errors.New("Görsel indirilemedi.") }
	if response.ContentLength > maxProductImageSize { return nil, "", errors.New("Görsel boyutu çok büyük.") }
	data, err := io.ReadAll(io.LimitReader(response.Body, maxProductImageSize+1))
	if err != nil { return nil, "", errors.New("Görsel indirilemedi.") }
	ext, err := validateDownloadedImage(data)
	if err != nil { return nil, "", err }
	return data, ext, nil
}

func (h *ProductHandler) ImportImage(c *gin.Context) {
	id, valid := parseID(c)
	if !valid { return }
	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil { handleDBError(c, err); return }
	var input struct { URL string `json:"url"` }
	if err := c.ShouldBindJSON(&input); err != nil { fail(c, http.StatusBadRequest, "Geçersiz görsel URL'si."); return }
	source := strings.TrimSpace(input.URL)
	if err := validateImageSource(source); err != nil { fail(c, http.StatusBadRequest, err.Error()); return }
	data, ext, err := downloadProductImage(source)
	if err != nil { fail(c, http.StatusBadRequest, err.Error()); return }
	if err := os.MkdirAll(productImageDir, 0755); err != nil { fail(c, http.StatusInternalServerError, "Görsel kaydedilemedi."); return }
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil { fail(c, http.StatusInternalServerError, "Görsel kaydedilemedi."); return }
	filename := fmt.Sprintf("product-%d-%s%s", id, hex.EncodeToString(random), ext)
	path := filepath.Join(productImageDir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil { fail(c, http.StatusInternalServerError, "Görsel kaydedilemedi."); return }
	oldURL := product.ImageURL
	newURL := "/api/product-images/" + filename
	if err := h.DB.Model(&product).Updates(map[string]any{"image_url": newURL, "image_source_url": source}).Error; err != nil {
		_ = os.Remove(path)
		handleDBError(c, err)
		return
	}
	if imageFileName.MatchString(strings.TrimPrefix(oldURL, "/api/product-images/")) && strings.HasPrefix(oldURL, "/api/product-images/") {
		var uses int64
		if h.DB.Model(&models.Product{}).Where("image_url = ?", oldURL).Count(&uses).Error == nil && uses == 0 {
			_ = os.Remove(filepath.Join(productImageDir, filepath.Base(oldURL)))
		}
	}
	result, err := h.toResponse(product)
	if err != nil { handleDBError(c, err); return }
	ok(c, result)
}

func ServeProductImage(c *gin.Context) {
	filename := c.Param("filename")
	if !imageFileName.MatchString(filename) { fail(c, http.StatusNotFound, "Görsel bulunamadı."); return }
	path := filepath.Join(productImageDir, filename)
	if _, err := os.Stat(path); err != nil { fail(c, http.StatusNotFound, "Görsel bulunamadı."); return }
	c.Header("Cache-Control", "public, max-age=86400")
	c.File(path)
}

func productImageExists(imageURL string) bool {
	const prefix = "/api/product-images/"
	if !strings.HasPrefix(imageURL, prefix) { return false }
	filename := strings.TrimPrefix(imageURL, prefix)
	if !imageFileName.MatchString(filename) { return false }
	info, err := os.Stat(filepath.Join(productImageDir, filename))
	return err == nil && info.Mode().IsRegular()
}
