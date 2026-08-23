package limiter

import (
	"context"
	"strings"
)

type Dim struct {
	Key        string
	RatePerMin float64
	Burst      float64
}

type Multi struct {
	AllowFn func(ctx context.Context, key string, rate, burst float64) (bool, float64, error)
}

// DomainClass maps recipient domain onto the isolated rate plane.
func DomainClass(emailOrDomain string) string {
	d := strings.ToLower(emailOrDomain)
	if i := strings.LastIndex(d, "@"); i >= 0 {
		d = d[i+1:]
	}
	switch d {
	case "gmail.com", "googlemail.com":
		return "gmail"
	case "outlook.com", "hotmail.com", "live.com", "msn.com", "office365.com":
		return "outlook"
	default:
		return "other"
	}
}

type Decision struct {
	Allowed bool
	Blocked string
	Remain  map[string]float64
}

func (m *Multi) AllowAll(ctx context.Context, dims []Dim) (Decision, error) {
	dec := Decision{Allowed: true, Remain: map[string]float64{}}
	for _, d := range dims {
		ok, left, err := m.AllowFn(ctx, d.Key, d.RatePerMin, d.Burst)
		if err != nil {
			return dec, err
		}
		dec.Remain[d.Key] = left
		if !ok {
			dec.Allowed = false
			dec.Blocked = d.Key
			return dec, nil
		}
	}
	return dec, nil
}
