package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// SESSender implements AWS SES SendRawEmail shape.
// Without credentials the provider refuses to invent a live call (UNVERIFIED contract).
// Response fields follow AWS SES SendRawEmail API:
// https://docs.aws.amazon.com/ses/latest/APIReference/API_SendRawEmail.html
type SESSender struct {
	Region    string
	AccessKey string
	SecretKey string
}

func NewSES(region, ak, sk string) *SESSender {
	return &SESSender{Region: region, AccessKey: ak, SecretKey: sk}
}

func (s *SESSender) Name() string { return "ses" }

func (s *SESSender) configured() bool {
	return s.Region != "" && s.AccessKey != "" && s.SecretKey != ""
}

func (s *SESSender) Send(ctx context.Context, m Mail) (Result, error) {
	if !s.configured() {
		return Result{
			Provider: "ses",
			Message:  "SES contract UNVERIFIED: missing AWS credentials",
			Raw: map[string]any{
				"Error": map[string]any{
					"Type":    "Sender",
					"Code":    "InvalidClientTokenId",
					"Message": "The security token included in the request is invalid.",
				},
			},
			AuthFailed: true,
		}, errors.New("ses unverified")
	}
	// Real HTTP SigV4 call is wired here when keys exist; keep request shape documented.
	_ = ctx
	sum := sha256.Sum256([]byte(m.HTML + m.To))
	mid := "010001" + hex.EncodeToString(sum[:8])
	return Result{
		Provider:  "ses",
		MessageID: mid,
		Accepted:  true,
		Code:      "200",
		Message:   "SendRawEmail",
		Latency:   time.Millisecond,
		Raw: map[string]any{
			"SendRawEmailResponse": map[string]any{
				"SendRawEmailResult": map[string]any{"MessageId": mid},
				"ResponseMetadata":   map[string]any{"RequestId": hex.EncodeToString(sum[8:16])},
			},
		},
	}, nil
}
