package bounce

import (
	"strings"

	"github.com/lumen/relay/internal/domain"
)

// Classify maps SMTP / DSN codes onto hard / soft / block.
// Enhanced codes follow RFC 3463.
func Classify(smtpCode, enhanced, message string) domain.BounceClass {
	e := strings.TrimSpace(enhanced)
	if e == "" {
		e = inferEnhanced(smtpCode, message)
	}
	if strings.HasPrefix(e, "5.1.") || strings.HasPrefix(e, "5.2.1") || e == "5.4.4" {
		return domain.BounceHard
	}
	if strings.HasPrefix(e, "5.7.") || strings.Contains(strings.ToLower(message), "blocked") ||
		strings.Contains(strings.ToLower(message), "spam") {
		return domain.BounceBlock
	}
	if strings.HasPrefix(e, "4.") || strings.HasPrefix(smtpCode, "4") {
		return domain.BounceSoft
	}
	if strings.HasPrefix(smtpCode, "5") {
		msg := strings.ToLower(message)
		if strings.Contains(msg, "user unknown") || strings.Contains(msg, "does not exist") ||
			strings.Contains(msg, "mailbox unavailable") || strings.Contains(msg, "no such") {
			return domain.BounceHard
		}
		return domain.BounceBlock
	}
	return domain.BounceOK
}

func inferEnhanced(code, message string) string {
	msg := strings.ToLower(message)
	switch code {
	case "550", "551", "553":
		if strings.Contains(msg, "user") || strings.Contains(msg, "exist") || strings.Contains(msg, "unknown") {
			return "5.1.1"
		}
		return "5.7.1"
	case "552":
		return "5.2.2"
	case "421", "450", "451", "452":
		return "4.2.1"
	case "535":
		return "5.7.8"
	}
	return ""
}
