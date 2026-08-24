package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lumen/relay/internal/bounce"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/limiter"
	"github.com/lumen/relay/internal/model"
	"github.com/lumen/relay/internal/pipeline"
	"github.com/lumen/relay/internal/provider"
	"github.com/lumen/relay/internal/render"
	"github.com/lumen/relay/internal/repo"
	"github.com/lumen/relay/internal/token"
	"gorm.io/datatypes"
)

type Campaigns struct {
	Store repo.Store
	Q     *pipeline.Queue
	Cfg   config.Config
	Send  provider.Sender
	Bal   *provider.Balancer
	Lim   *limiter.Multi
}

type LaunchReq struct {
	Name            string     `json:"name"`
	FromName        string     `json:"from_name"`
	FromEmail       string     `json:"from_email"`
	ReplyTo         string     `json:"reply_to"`
	Subject         string     `json:"subject"`
	ListID          string     `json:"list_id"`
	TemplateVerID   string     `json:"template_ver_id"`
	Strategy        string     `json:"strategy"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	BatchSize       int        `json:"batch_size"`
	BatchIntervalS  int        `json:"batch_interval_s"`
	RampPercent     int        `json:"ramp_percent"`
}

func (s Campaigns) Create(tenant string, req LaunchReq) (*model.Campaign, error) {
	if req.Name == "" || req.FromEmail == "" || req.Subject == "" {
		return nil, fmt.Errorf("%w: name/from/subject required", domain.ErrValidation)
	}
	if req.Strategy == "" {
		req.Strategy = "immediate"
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 200
	}
	if req.BatchIntervalS <= 0 {
		req.BatchIntervalS = 60
	}
	c := &model.Campaign{
		ID: uuid.NewString(), TenantID: tenant, Name: req.Name, Status: "draft",
		FromName: req.FromName, FromEmail: req.FromEmail, ReplyTo: req.ReplyTo,
		Subject: req.Subject, ListID: req.ListID, TemplateVerID: req.TemplateVerID,
		Strategy: req.Strategy, ScheduledAt: req.ScheduledAt,
		BatchSize: req.BatchSize, BatchIntervalS: req.BatchIntervalS, RampPercent: req.RampPercent,
		CreatedAt: clock.Now(), UpdatedAt: clock.Now(),
	}
	return c, s.Store.DB.Create(c).Error
}

func (s Campaigns) Transit(tenant, id, to, reason string) (*model.Campaign, error) {
	c, err := s.Store.Campaign(tenant, id)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransitCampaign(c.Status, to) {
		return nil, fmt.Errorf("%w: %s → %s", domain.ErrInvalidState, c.Status, to)
	}
	c.Status = to
	c.PausedReason = reason
	c.UpdatedAt = clock.Now()
	if err := s.Store.DB.Save(c).Error; err != nil {
		return nil, err
	}
	if to == "running" {
		return c, s.materializeAndEnqueue(c)
	}
	return c, nil
}

func (s Campaigns) materializeAndEnqueue(c *model.Campaign) error {
	contacts, err := s.Store.ListMembers(c.TenantID, c.ListID)
	if err != nil {
		return err
	}
	now := clock.Now()
	var jobs []domain.SendJob
	for _, ct := range contacts {
		if s.Store.IsSuppressed(c.TenantID, ct.Email) {
			rc := model.CampaignRecipient{
				ID: uuid.NewString(), TenantID: c.TenantID, CampaignID: c.ID, ContactID: ct.ID,
				Email: ct.Email, Domain: domainOf(ct.Email), Status: "skipped",
				CreatedAt: now, UpdatedAt: now,
			}
			_ = s.Store.DB.Create(&rc).Error
			_ = s.Store.AppendEvent(model.EmailEvent{
				TenantID: c.TenantID, CampaignID: c.ID, RecipientID: rc.ID, Kind: "skip",
			})
			continue
		}
		rc := model.CampaignRecipient{
			ID: uuid.NewString(), TenantID: c.TenantID, CampaignID: c.ID, ContactID: ct.ID,
			Email: ct.Email, Domain: domainOf(ct.Email), Status: "pending",
			MessageID: uuid.NewString(), CreatedAt: now, UpdatedAt: now,
		}
		if err := s.Store.DB.Create(&rc).Error; err != nil {
			continue
		}
		jobs = append(jobs, domain.SendJob{
			RecipientID: rc.ID, CampaignID: c.ID, TenantID: c.TenantID,
			DomainHint: limiter.DomainClass(ct.Email),
		})
	}
	if c.Strategy == "throttled" && c.BatchSize > 0 && len(jobs) > c.BatchSize {
		first := jobs[:c.BatchSize]
		rest := jobs[c.BatchSize:]
		if err := s.Q.PushMany(context.Background(), first); err != nil {
			return err
		}
		at := now.Add(time.Duration(c.BatchIntervalS) * time.Second)
		for i, j := range rest {
			delay := at.Add(time.Duration(i/c.BatchSize) * time.Duration(c.BatchIntervalS) * time.Second)
			_ = s.Q.Delay(context.Background(), j, delay)
		}
		return nil
	}
	return s.Q.PushMany(context.Background(), jobs)
}

func (s Campaigns) HandleJob(ctx context.Context, job domain.SendJob) error {
	rc, err := s.Store.Recipient(job.RecipientID)
	if err != nil {
		return err
	}
	if rc.Status == "sent" || rc.Status == "delivered" || rc.Status == "skipped" || rc.Status == "bounced" {
		return nil
	}
	c, err := s.Store.Campaign(rc.TenantID, rc.CampaignID)
	if err != nil {
		return err
	}
	if c.Status == "paused" || c.Status == "cancelled" {
		return s.Q.Delay(ctx, job, clock.Now().Add(5*time.Second))
	}
	if s.Store.IsSuppressed(rc.TenantID, rc.Email) {
		rc.Status = "skipped"
		rc.UpdatedAt = clock.Now()
		_ = s.Store.DB.Save(rc).Error
		return s.Store.AppendEvent(model.EmailEvent{TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "skip"})
	}

	dims := s.rateDims(rc)
	dec, err := s.Lim.AllowAll(ctx, dims)
	if err != nil {
		return err
	}
	if !dec.Allowed {
		return s.Q.Delay(ctx, job, clock.Now().Add(2*time.Second))
	}

	chKey := "smtp-default"
	if s.Bal != nil {
		if h, ok := s.Bal.Pick(); ok {
			chKey = h.Key
		}
	}
	rc.Status = "sending"
	rc.ChannelKey = chKey
	rc.Attempt++
	rc.UpdatedAt = clock.Now()
	_ = s.Store.DB.Save(rc).Error

	htmlBody, unsubURL, err := s.renderOne(c, rc)
	if err != nil {
		return s.fail(ctx, job, rc, err.Error(), false)
	}
	mid := rc.MessageID
	claimed, _ := s.Q.ClaimIdem(ctx, mid, 24*time.Hour)
	if !claimed && mid != "" {
		return nil
	}

	res, sendErr := s.Send.Send(ctx, provider.Mail{
		MessageID: mid, FromName: c.FromName, FromEmail: c.FromEmail, ReplyTo: c.ReplyTo,
		To: rc.Email, Subject: applySubject(c.Subject, rc), HTML: htmlBody.HTML, Text: htmlBody.Text,
		ChannelKey: chKey, UnsubURL: unsubURL, UnsubMailto: "unsub+" + rc.ID + "@lumen.local",
		Headers: map[string]string{"X-Lumen-Campaign": c.ID},
	})
	if s.Bal != nil {
		s.Bal.Report(chKey, res.Accepted)
	}
	if res.Accepted {
		now := clock.Now()
		rc.Status = "sent"
		rc.SentAt = &now
		rc.UpdatedAt = now
		_ = s.Store.DB.Save(rc).Error
		_ = s.Store.AppendEvent(model.EmailEvent{TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "sent"})
		_ = s.Store.AppendEvent(model.EmailEvent{TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "delivered"})
		s.maybeComplete(c)
		s.maybePauseReputation(c)
		return nil
	}

	class := bounce.Classify(res.Code, res.Enhanced, res.Message)
	if class != domain.BounceOK {
		s.applyBounce(rc, domain.BounceEvent{
			Email: rc.Email, Class: class, Code: res.Code, Enhanced: res.Enhanced,
			Message: res.Message, CampaignID: rc.CampaignID, TenantID: rc.TenantID, Source: "smtp",
		})
	}
	retry, wait := pipeline.ShouldRetry(res, sendErr, rc.Attempt)
	if retry {
		job.Attempt = rc.Attempt
		rc.Status = "queued"
		rc.LastError = res.Message
		rc.UpdatedAt = clock.Now()
		_ = s.Store.DB.Save(rc).Error
		return s.Q.Delay(ctx, job, clock.Now().Add(wait))
	}
	return s.fail(ctx, job, rc, res.Message, true)
}

func (s Campaigns) fail(ctx context.Context, job domain.SendJob, rc *model.CampaignRecipient, msg string, dead bool) error {
	rc.Status = "failed"
	rc.LastError = msg
	rc.UpdatedAt = clock.Now()
	_ = s.Store.DB.Save(rc).Error
	_ = s.Store.AppendEvent(model.EmailEvent{TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "fail"})
	if dead {
		_ = s.Q.Dead(ctx, job)
	}
	return nil
}

func (s Campaigns) applyBounce(rc *model.CampaignRecipient, ev domain.BounceEvent) {
	var ct model.Contact
	_ = s.Store.DB.First(&ct, "id = ?", rc.ContactID).Error
	act := bounce.Next(ct.Status, ev, ct.SoftBounce)
	if act.IncrementSoft {
		ct.SoftBounce++
	}
	if act.ContactStatus != "" {
		ct.Status = act.ContactStatus
	}
	ct.UpdatedAt = clock.Now()
	_ = s.Store.DB.Save(&ct).Error
	if act.Suppress {
		_ = s.Store.Suppress(rc.TenantID, rc.Email, act.Reason, ev.Source, ev.Message)
		rc.Status = "bounced"
		rc.UpdatedAt = clock.Now()
		_ = s.Store.DB.Save(rc).Error
	}
	_ = s.Store.DB.Create(&model.BounceRecord{
		ID: uuid.NewString(), TenantID: rc.TenantID, Email: rc.Email, Class: string(ev.Class),
		Code: ev.Code, Enhanced: ev.Enhanced, Message: ev.Message, Source: ev.Source, CreatedAt: clock.Now(),
	}).Error
	_ = s.Store.AppendEvent(model.EmailEvent{TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "bounce"})
}

func (s Campaigns) renderOne(c *model.Campaign, rc *model.CampaignRecipient) (render.Result, string, error) {
	ver, err := s.Store.TemplateVersion(c.TenantID, c.TemplateVerID)
	if err != nil {
		return render.Result{}, "", err
	}
	var ast domain.TemplateAST
	if err := json.Unmarshal(ver.AST, &ast); err != nil {
		return render.Result{}, "", err
	}
	vars := map[string]string{"UserName": "", "Email": rc.Email}
	var ct model.Contact
	if err := s.Store.DB.First(&ct, "id = ?", rc.ContactID).Error; err == nil {
		vars["UserName"] = ct.Name
		var attrs map[string]string
		if len(ct.Attrs) > 0 {
			_ = json.Unmarshal(ct.Attrs, &attrs)
			for k, v := range attrs {
				vars[k] = v
			}
		}
	}
	unsub := s.Cfg.PublicUnsubBase + "/u?t=" + url.QueryEscape(token.UnsubToken(s.Cfg.TrackHMAC, c.TenantID, c.ID, rc.ID))
	out, err := render.Engine(ast, render.Context{
		Vars: vars, Secret: s.Cfg.TrackHMAC, TrackBase: s.Cfg.PublicTrackBase,
		UnsubBase: s.Cfg.PublicUnsubBase, TenantID: c.TenantID, CampaignID: c.ID,
		RecipientID: rc.ID, IncludePixel: true, RewriteLinks: true,
	})
	return out, unsub, err
}

func (s Campaigns) rateDims(rc *model.CampaignRecipient) []limiter.Dim {
	cls := limiter.DomainClass(rc.Email)
	rate := float64(s.Cfg.RateOtherPerMin)
	switch cls {
	case "gmail":
		rate = float64(s.Cfg.RateGmailPerMin)
	case "outlook":
		rate = float64(s.Cfg.RateOutlookPerMin)
	}
	return []limiter.Dim{
		{Key: "dom:" + cls, RatePerMin: rate, Burst: rate},
		{Key: "ch:" + rc.ChannelKey, RatePerMin: float64(s.Cfg.RateChannelPerMin), Burst: float64(s.Cfg.RateChannelPerMin)},
		{Key: "ten:" + rc.TenantID, RatePerMin: float64(s.Cfg.RateTenantPerMin), Burst: float64(s.Cfg.RateTenantPerMin)},
	}
}

func (s Campaigns) maybeComplete(c *model.Campaign) {
	var left int64
	s.Store.DB.Model(&model.CampaignRecipient{}).
		Where("campaign_id = ? AND status IN ?", c.ID, []string{"pending", "queued", "sending"}).
		Count(&left)
	if left == 0 {
		c.Status = "completed"
		c.UpdatedAt = clock.Now()
		_ = s.Store.DB.Save(c).Error
	}
}

func (s Campaigns) maybePauseReputation(c *model.Campaign) {
	hard, comp, sent, err := s.Store.Reputation(c.TenantID, c.ID)
	if err != nil || sent < 50 {
		return
	}
	if hard > 0.05 || comp > 0.003 {
		c.Status = "paused"
		c.PausedReason = "reputation_redline"
		c.UpdatedAt = clock.Now()
		_ = s.Store.DB.Save(c).Error
	}
}

func (s Campaigns) RecordOpen(p token.Payload, ua, ip string) {
	kind := "open"
	if isMachineOpen(ua, ip) {
		kind = "machine_open"
	}
	uniq := !s.Store.HasUnique(p.TenantID, p.Campaign, p.Recipient, kind)
	meta, _ := json.Marshal(map[string]string{"ua": ua, "ip": ip})
	_ = s.Store.AppendEvent(model.EmailEvent{
		TenantID: p.TenantID, CampaignID: p.Campaign, RecipientID: p.Recipient,
		Kind: kind, UniqueFlag: uniq, Meta: datatypes.JSON(meta),
	})
}

func (s Campaigns) RecordClick(p token.Payload) {
	uniq := !s.Store.HasUnique(p.TenantID, p.Campaign, p.Recipient, "click")
	_ = s.Store.AppendEvent(model.EmailEvent{
		TenantID: p.TenantID, CampaignID: p.Campaign, RecipientID: p.Recipient,
		Kind: "click", UniqueFlag: uniq, URL: p.URL,
	})
}

func (s Campaigns) Unsubscribe(p token.Payload) error {
	rc, err := s.Store.Recipient(p.Recipient)
	if err != nil {
		return err
	}
	act := bounce.OnUnsubscribe()
	_ = s.Store.DB.Model(&model.Contact{}).Where("id = ?", rc.ContactID).Updates(map[string]any{
		"status": act.ContactStatus, "updated_at": clock.Now(),
	}).Error
	_ = s.Store.Suppress(rc.TenantID, rc.Email, act.Reason, "unsub", "")
	_ = s.Store.AppendEvent(model.EmailEvent{
		TenantID: rc.TenantID, CampaignID: rc.CampaignID, RecipientID: rc.ID, Kind: "unsub", UniqueFlag: true,
	})
	return nil
}

func domainOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return strings.ToLower(email[i+1:])
	}
	return email
}

func applySubject(subj string, rc *model.CampaignRecipient) string {
	return strings.ReplaceAll(subj, "{{ .UserName }}", rc.Email)
}

func isMachineOpen(ua, ip string) bool {
	u := strings.ToLower(ua)
	if strings.Contains(u, "googleimageproxy") || strings.Contains(u, "ggpht.com") ||
		strings.Contains(u, "yahoo mail proxy") || strings.Contains(u, "apple") && strings.Contains(u, "mail") {
		return true
	}
	_ = ip
	return false
}
