package tracker

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/httpx"
	"github.com/lumen/relay/internal/service"
	"github.com/lumen/relay/internal/token"
)

// 1x1 transparent GIF
var gif = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

type Gateway struct {
	Cfg  config.Config
	Log  *slog.Logger
	Camp service.Campaigns
}

func (g Gateway) Engine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), httpx.Trace())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "role": "tracker", "ts": clock.Format(clock.Now())})
	})
	r.GET("/o/*token", g.open)
	r.GET("/c/*token", g.click)
	return r
}

func (g Gateway) open(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("token"), "/")
	raw = strings.TrimSuffix(raw, ".gif")
	// respond first — never block pixel on DB
	c.Header("Cache-Control", "no-store, private")
	c.Data(http.StatusOK, "image/gif", gif)
	p, err := token.Verify(g.Cfg.TrackHMAC, raw)
	if err != nil || p.Kind != "o" {
		return
	}
	go g.Camp.RecordOpen(p, c.GetHeader("User-Agent"), c.ClientIP())
}

func (g Gateway) click(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("token"), "/")
	p, err := token.Verify(g.Cfg.TrackHMAC, raw)
	dest := "https://example.com"
	if err == nil && p.Kind == "c" && looksSafe(p.URL) {
		dest = p.URL
		go g.Camp.RecordClick(p)
	}
	c.Redirect(http.StatusFound, dest)
}

func looksSafe(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
