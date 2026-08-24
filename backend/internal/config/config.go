package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Role             string
	Env              string
	LogLevel         string
	HTTPAddr         string
	DatabaseURL      string
	RedisAddr        string
	JWTSecret        string
	JWTAccessMin     int
	JWTRefreshH      int
	TrackHMAC        string
	PublicTrackBase  string
	PublicUnsubBase  string
	MailProvider     string
	SMTPHost         string
	SMTPPort         int
	SMTPUser         string
	SMTPPass         string
	SMTPStartTLS     bool
	BounceSource     string
	AWSRegion        string
	AWSAccessKey     string
	AWSSecretKey     string
	SenderWorkers    int
	RateGmailPerMin  int
	RateOutlookPerMin int
	RateOtherPerMin  int
	RateChannelPerMin int
	RateTenantPerMin int
	CORSOrigins      []string
}

func Load() Config {
	c := Config{
		Role:              env("APP_ROLE", "api"),
		Env:               env("APP_ENV", "dev"),
		LogLevel:          env("LOG_LEVEL", "info"),
		HTTPAddr:          env("HTTP_ADDR", ":8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://lumen:lumen_dev_pw@127.0.0.1:5432/lumen?sslmode=disable"),
		RedisAddr:         env("REDIS_ADDR", "127.0.0.1:6379"),
		JWTSecret:         env("JWT_SECRET", "lumen-relay-dev-secret-change-me"),
		JWTAccessMin:      envInt("JWT_ACCESS_TTL_MIN", 120),
		JWTRefreshH:       envInt("JWT_REFRESH_TTL_H", 168),
		TrackHMAC:         env("TRACK_HMAC_SECRET", "lumen-track-hmac-dev-change-me"),
		PublicTrackBase:   strings.TrimRight(env("PUBLIC_TRACK_BASE", "http://localhost:27483"), "/"),
		PublicUnsubBase:   strings.TrimRight(env("PUBLIC_UNSUB_BASE", "http://localhost:27485"), "/"),
		MailProvider:      strings.ToLower(env("MAIL_PROVIDER", "smtp")),
		SMTPHost:          env("SMTP_HOST", "mailpit"),
		SMTPPort:          envInt("SMTP_PORT", 1025),
		SMTPUser:          env("SMTP_USER", ""),
		SMTPPass:          env("SMTP_PASS", ""),
		SMTPStartTLS:      envBool("SMTP_STARTTLS", false),
		BounceSource:      strings.ToLower(env("BOUNCE_SOURCE", "smtp")),
		AWSRegion:         env("AWS_REGION", ""),
		AWSAccessKey:      env("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:      env("AWS_SECRET_ACCESS_KEY", ""),
		SenderWorkers:     envInt("SENDER_WORKERS", 64),
		RateGmailPerMin:   envInt("RATE_GMAIL_PER_MIN", 120),
		RateOutlookPerMin: envInt("RATE_OUTLOOK_PER_MIN", 180),
		RateOtherPerMin:   envInt("RATE_OTHER_PER_MIN", 300),
		RateChannelPerMin: envInt("RATE_CHANNEL_PER_MIN", 600),
		RateTenantPerMin:  envInt("RATE_TENANT_PER_MIN", 1000),
	}
	raw := env("CORS_ORIGINS", "http://localhost:27481,http://localhost:27485")
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(p); s != "" {
			c.CORSOrigins = append(c.CORSOrigins, s)
		}
	}
	return c
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
