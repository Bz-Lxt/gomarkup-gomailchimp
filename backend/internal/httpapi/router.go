package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/httpx"
	"github.com/lumen/relay/internal/model"
	"github.com/lumen/relay/internal/pipeline"
	"github.com/lumen/relay/internal/repo"
	"github.com/lumen/relay/internal/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Deps struct {
	Cfg   config.Config
	Log   *slog.Logger
	DB    *gorm.DB
	RDB   *redis.Client
	Store repo.Store
	Auth  service.Auth
	Imp   service.Importer
	Camp  service.Campaigns
	Q     *pipeline.Queue
}

func New(d Deps) *gin.Engine {
	if d.Cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), httpx.Trace(), httpx.AccessLog(d.Log))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     d.Cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
	}))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true, "role": "api", "ts": clock.Format(clock.Now())}) })
	r.GET("/readyz", func(c *gin.Context) {
		// Derive the timeout from the request context so that when the probe
		// client gives up and disconnects, in-flight dependency checks are
		// cancelled immediately instead of hanging until the server-side
		// timeout. The 2 s deadline still acts as a server-side backstop.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := repo.Ping(ctx, d.DB); err != nil {
			c.JSON(503, gin.H{"ok": false, "db": err.Error()})
			return
		}
		if err := d.RDB.Ping(ctx).Err(); err != nil {
			c.JSON(503, gin.H{"ok": false, "redis": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	h := handler{d}
	api := r.Group("/api/v1")
	api.POST("/auth/login", h.login)
	api.POST("/auth/refresh", h.refresh)
	api.POST("/public/unsub", h.publicUnsub)
	api.POST("/public/unsub/one-click", h.oneClick)

	authed := api.Group("", httpx.JWT(d.Cfg.JWTSecret))
	authed.GET("/me", h.me)
	authed.GET("/contacts", h.listContacts)
	authed.GET("/lists", h.listLists)
	authed.POST("/lists", httpx.RequireWrite(), h.createList)
	authed.POST("/contacts/import", httpx.RequireWrite(), h.importContacts)
	authed.GET("/templates", h.listTemplates)
	authed.POST("/templates", httpx.RequireWrite(), h.saveTemplate)
	authed.GET("/templates/:id", h.getTemplate)
	authed.GET("/campaigns", h.listCampaigns)
	authed.POST("/campaigns", httpx.RequireWrite(), h.createCampaign)
	authed.GET("/campaigns/:id", h.getCampaign)
	authed.POST("/campaigns/:id/action", httpx.RequireWrite(), h.campaignAction)
	authed.GET("/campaigns/:id/funnel", h.funnel)
	authed.GET("/campaigns/:id/stream", h.stream)
	authed.GET("/channels", h.channels)
	authed.GET("/suppressions", h.suppressions)
	authed.GET("/pipeline/stats", h.pipeStats)
	return r
}

type handler struct{ Deps }

func (h handler) login(c *gin.Context) {
	var req struct{ Email, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	tok, cl, err := h.Auth.Login(req.Email, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"tokens": tok, "user": gin.H{"id": cl.UserID, "email": cl.Email, "role": cl.Role, "tenant_id": cl.TenantID}})
}

func (h handler) refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	tok, cl, err := h.Auth.Refresh(req.RefreshToken)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"tokens": tok, "user": cl})
}

func (h handler) me(c *gin.Context) {
	cl := httpx.Claims(c)
	t, _ := h.Store.Tenant(cl.TenantID)
	httpx.OK(c, gin.H{"user": cl, "tenant": t})
}

func (h handler) listContacts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	per, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if per < 1 || per > 200 {
		per = 20
	}
	rows, total, err := h.Store.ListContacts(httpx.Claims(c).TenantID, c.Query("q"), page, per)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Page(c, rows, total, page, per)
}

func (h handler) listLists(c *gin.Context) {
	rows, err := h.Store.Lists(httpx.Claims(c).TenantID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h handler) createList(c *gin.Context) {
	var req struct{ Name string `json:"name"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	row := model.ContactList{ID: uuid.NewString(), TenantID: httpx.Claims(c).TenantID, Name: req.Name, CreatedAt: clock.Now()}
	if err := h.DB.Create(&row).Error; err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, row)
}

func (h handler) importContacts(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	defer f.Close()
	peek := make([]byte, 8)
	n, _ := io.ReadFull(f, peek)
	peek = peek[:n]
	res, err := h.Imp.Import(httpx.Claims(c).TenantID, c.PostForm("list_id"), fh.Filename, f, peek)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h handler) listTemplates(c *gin.Context) {
	var rows []model.Template
	if err := repo.WithTenant(h.DB, httpx.Claims(c).TenantID).Order("updated_at DESC").Find(&rows).Error; err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h handler) saveTemplate(c *gin.Context) {
	var req struct {
		ID      string             `json:"id"`
		Name    string             `json:"name"`
		Subject string             `json:"subject"`
		AST     domain.TemplateAST `json:"ast"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	tid := httpx.Claims(c).TenantID
	raw, _ := json.Marshal(req.AST)
	now := clock.Now()
	tpl := model.Template{ID: req.ID, TenantID: tid, Name: req.Name, CreatedAt: now, UpdatedAt: now}
	if tpl.ID == "" {
		tpl.ID = uuid.NewString()
		if err := h.DB.Create(&tpl).Error; err != nil {
			httpx.Fail(c, err)
			return
		}
	} else {
		if err := repo.WithTenant(h.DB, tid).First(&model.Template{}, "id = ?", tpl.ID).Error; err != nil {
			httpx.Fail(c, domain.ErrNotFound)
			return
		}
		h.DB.Model(&model.Template{}).Where("id = ?", tpl.ID).Updates(map[string]any{"name": req.Name, "updated_at": now})
	}
	var last model.TemplateVersion
	ver := 1
	if err := h.DB.Where("template_id = ?", tpl.ID).Order("version DESC").First(&last).Error; err == nil {
		ver = last.Version + 1
	}
	tv := model.TemplateVersion{
		ID: uuid.NewString(), TenantID: tid, TemplateID: tpl.ID, Version: ver,
		Subject: req.Subject, AST: raw, CreatedAt: now,
	}
	if err := h.DB.Create(&tv).Error; err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, gin.H{"template": tpl, "version": tv})
}

func (h handler) getTemplate(c *gin.Context) {
	tid := httpx.Claims(c).TenantID
	var tpl model.Template
	if err := repo.WithTenant(h.DB, tid).First(&tpl, "id = ?", c.Param("id")).Error; err != nil {
		httpx.Fail(c, domain.ErrNotFound)
		return
	}
	var ver model.TemplateVersion
	_ = h.DB.Where("template_id = ?", tpl.ID).Order("version DESC").First(&ver).Error
	httpx.OK(c, gin.H{"template": tpl, "version": ver})
}

func (h handler) listCampaigns(c *gin.Context) {
	var rows []model.Campaign
	if err := repo.WithTenant(h.DB, httpx.Claims(c).TenantID).Order("created_at DESC").Find(&rows).Error; err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h handler) createCampaign(c *gin.Context) {
	var req service.LaunchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	row, err := h.Camp.Create(httpx.Claims(c).TenantID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, row)
}

func (h handler) getCampaign(c *gin.Context) {
	row, err := h.Store.Campaign(httpx.Claims(c).TenantID, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h handler) campaignAction(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	to := map[string]string{"start": "running", "schedule": "scheduled", "pause": "paused", "resume": "running", "cancel": "cancelled"}[req.Action]
	if to == "" {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	row, err := h.Camp.Transit(httpx.Claims(c).TenantID, c.Param("id"), to, "")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h handler) funnel(c *gin.Context) {
	snap, err := h.Store.Funnel(httpx.Claims(c).TenantID, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, snap)
}

func (h handler) stream(c *gin.Context) {
	cl := httpx.Claims(c)
	id := c.Param("id")
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	tick := time.NewTicker(1500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-tick.C:
			snap, err := h.Store.Funnel(cl.TenantID, id)
			if err != nil {
				return
			}
			b, _ := json.Marshal(snap)
			_, _ = c.Writer.Write([]byte("event: funnel\ndata: " + string(b) + "\n\n"))
			c.Writer.Flush()
		}
	}
}

func (h handler) channels(c *gin.Context) {
	rows, err := h.Store.Channels(httpx.Claims(c).TenantID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h handler) suppressions(c *gin.Context) {
	var rows []model.Suppression
	if err := repo.WithTenant(h.DB, httpx.Claims(c).TenantID).Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h handler) pipeStats(c *gin.Context) {
	send, delay, dlq, err := h.Q.Depth(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"send": send, "delay": delay, "dlq": dlq, "provider": h.Cfg.MailProvider})
}

func (h handler) publicUnsub(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, domain.ErrValidation)
		return
	}
	p, err := parseUnsub(h.Cfg.TrackHMAC, req.Token)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.Camp.Unsubscribe(p); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h handler) oneClick(c *gin.Context) {
	_ = c.Request.ParseForm()
	tok := c.PostForm("token")
	if tok == "" {
		tok = c.Query("t")
	}
	p, err := parseUnsub(h.Cfg.TrackHMAC, tok)
	if err != nil {
		c.Status(http.StatusUnprocessableEntity)
		return
	}
	_ = h.Camp.Unsubscribe(p)
	c.Status(http.StatusOK)
}
