package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lumen/relay/internal/domain"
)

type Payload struct {
	Kind      string `json:"k"` // o | c | u
	TenantID  string `json:"t"`
	Campaign  string `json:"c"`
	Recipient string `json:"r"`
	URL       string `json:"u,omitempty"`
}

func Sign(secret string, p Payload) string {
	raw, _ := json.Marshal(p)
	body := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig
}

func Verify(secret, token string) (Payload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Payload{}, domain.ErrInvalidToken
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(want, got) {
		return Payload{}, domain.ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Payload{}, domain.ErrInvalidToken
	}
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Payload{}, domain.ErrInvalidToken
	}
	if p.Kind == "" || p.TenantID == "" || p.Recipient == "" {
		return Payload{}, domain.ErrInvalidToken
	}
	return p, nil
}

func OpenPath(secret string, tenant, campaign, recipient string) string {
	return fmt.Sprintf("/o/%s.gif", Sign(secret, Payload{
		Kind: "o", TenantID: tenant, Campaign: campaign, Recipient: recipient,
	}))
}

func ClickPath(secret, tenant, campaign, recipient, dest string) string {
	return fmt.Sprintf("/c/%s", Sign(secret, Payload{
		Kind: "c", TenantID: tenant, Campaign: campaign, Recipient: recipient, URL: dest,
	}))
}

func UnsubToken(secret, tenant, campaign, recipient string) string {
	return Sign(secret, Payload{Kind: "u", TenantID: tenant, Campaign: campaign, Recipient: recipient})
}
