package httpapi

import "github.com/lumen/relay/internal/token"

func parseUnsub(secret, raw string) (token.Payload, error) {
	return token.Verify(secret, raw)
}
