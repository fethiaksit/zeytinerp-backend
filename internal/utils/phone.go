package utils

import (
	"regexp"
	"strings"
)

var nonDigitRegexp = regexp.MustCompile(`[^\d]`)

// NormalizePhone converts phone numbers in various formats (e.g. 05321234567, 5321234567, +905321234567)
// into a standardized format "+905XXXXXXXXX".
func NormalizePhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return ""
	}

	// Remove all non-digits
	digits := nonDigitRegexp.ReplaceAllString(trimmed, "")
	if digits == "" {
		return ""
	}

	// Turkish 12-digit format with country code (905XXXXXXXXX)
	if len(digits) == 12 && strings.HasPrefix(digits, "90") {
		digits = digits[2:]
	} else if len(digits) == 11 && strings.HasPrefix(digits, "0") { // Turkish 11-digit format (05XXXXXXXXX)
		digits = digits[1:]
	}

	// If 10 digits starting with 5 (standard Turkish mobile number)
	if len(digits) == 10 && strings.HasPrefix(digits, "5") {
		return "+90" + digits
	}

	// Fallback for other formats (e.g. international)
	if strings.HasPrefix(trimmed, "+") {
		return "+" + digits
	}
	return digits
}

// PhoneVariants returns common representation variants of a phone number
// for SQL queries matching legacy databases.
func PhoneVariants(phone string) []string {
	norm := NormalizePhone(phone)
	if norm == "" {
		return []string{phone}
	}

	variants := []string{norm} // e.g. +905321234567
	if strings.HasPrefix(norm, "+90") && len(norm) == 13 {
		raw10 := norm[3:] // 5321234567
		raw11 := "0" + raw10 // 05321234567
		raw12 := "90" + raw10 // 905321234567
		variants = append(variants, raw11, raw10, raw12)
	}

	return variants
}
