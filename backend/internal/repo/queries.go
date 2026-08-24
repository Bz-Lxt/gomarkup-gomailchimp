package repo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct{ DB *gorm.DB }

func (s Store) Tenant(id string) (*model.Tenant, error) {
	var t model.Tenant
	err := s.DB.First(&t, "id = ?", id).Error
	return &t, mapErr(err)
}

func (s Store) UserByEmail(email string) (*model.User, error) {
	var u model.User
	err := s.DB.Where("email = ?", strings.ToLower(email)).First(&u).Error
	return &u, mapErr(err)
}

func (s Store) FindContact(tenant, email string) (*model.Contact, error) {
	var c model.Contact
	err := WithTenant(s.DB, tenant).Where("email = ?", strings.ToLower(email)).First(&c).Error
	return &c, mapErr(err)
}

func (s Store) UpsertContact(c *model.Contact) (created bool, err error) {
	c.Email = strings.ToLower(c.Email)
	var exist model.Contact
	q := WithTenant(s.DB, c.TenantID).Where("email = ?", c.Email).First(&exist)
	if q.Error == nil {
		exist.Name = c.Name
		if len(c.Attrs) > 0 {
			exist.Attrs = c.Attrs
		}
		exist.UpdatedAt = clock.Now()
		return false, s.DB.Save(&exist).Error
	}
	if q.Error != gorm.ErrRecordNotFound {
		return false, q.Error
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.CreatedAt = clock.Now()
	c.UpdatedAt = c.CreatedAt
	return true, s.DB.Create(c).Error
}

func (s Store) ListContacts(tenant, q string, page, per int) ([]model.Contact, int64, error) {
	db := WithTenant(s.DB, tenant).Model(&model.Contact{})
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("email ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Contact
	err := db.Order("created_at DESC").Limit(per).Offset((page - 1) * per).Find(&rows).Error
	return rows, total, err
}

func (s Store) Lists(tenant string) ([]model.ContactList, error) {
	var rows []model.ContactList
	err := WithTenant(s.DB, tenant).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (s Store) AddMembers(tenant, listID string, contactIDs []string) error {
	now := clock.Now()
	for _, id := range contactIDs {
		m := model.ListMembership{ListID: listID, ContactID: id, TenantID: tenant, CreatedAt: now}
		if err := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListMembers(tenant, listID string) ([]model.Contact, error) {
	var rows []model.Contact
	err := s.DB.Raw(`
		SELECT c.* FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id
		WHERE c.tenant_id = $1 AND m.list_id = $2 AND c.status = 'subscribed'
	`, tenant, listID).Scan(&rows).Error
	return rows, err
}

func (s Store) IsSuppressed(tenant, email string) bool {
	var n int64
	s.DB.Model(&model.Suppression{}).Where("tenant_id = ? AND email = ?", tenant, strings.ToLower(email)).Count(&n)
	return n > 0
}

func (s Store) Suppress(tenant, email, reason, source, detail string) error {
	row := model.Suppression{
		ID: uuid.NewString(), TenantID: tenant, Email: strings.ToLower(email),
		Reason: reason, Source: source, Detail: detail, CreatedAt: clock.Now(),
	}
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (s Store) Campaign(tenant, id string) (*model.Campaign, error) {
	var c model.Campaign
	err := WithTenant(s.DB, tenant).First(&c, "id = ?", id).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &c, nil
}

func (s Store) Recipient(id string) (*model.CampaignRecipient, error) {
	var r model.CampaignRecipient
	err := s.DB.First(&r, "id = ?", id).Error
	return &r, mapErr(err)
}

func (s Store) TemplateVersion(tenant, id string) (*model.TemplateVersion, error) {
	var v model.TemplateVersion
	err := WithTenant(s.DB, tenant).First(&v, "id = ?", id).Error
	return &v, mapErr(err)
}

func (s Store) Channels(tenant string) ([]model.SendChannel, error) {
	var rows []model.SendChannel
	err := WithTenant(s.DB, tenant).Find(&rows).Error
	return rows, err
}

func (s Store) AppendEvent(e model.EmailEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = clock.Now()
	}
	return s.DB.Create(&e).Error
}

func (s Store) HasUnique(tenant, campaign, recipient, kind string) bool {
	var n int64
	s.DB.Model(&model.EmailEvent{}).
		Where("tenant_id = ? AND campaign_id = ? AND recipient_id = ? AND kind = ? AND unique_flag = true",
			tenant, campaign, recipient, kind).Count(&n)
	return n > 0
}

func (s Store) Funnel(tenant, campaign string) (domain.FunnelSnapshot, error) {
	var snap domain.FunnelSnapshot
	snap.CampaignID = campaign
	type row struct {
		Kind  string
		Cnt   int64
		Uniq  int64
	}
	var rows []row
	err := s.DB.Raw(`
		SELECT kind,
		       COUNT(*) AS cnt,
		       COUNT(*) FILTER (WHERE unique_flag) AS uniq
		FROM email_events
		WHERE tenant_id = $1 AND campaign_id = $2
		GROUP BY kind
	`, tenant, campaign).Scan(&rows).Error
	if err != nil {
		return snap, err
	}
	for _, r := range rows {
		switch r.Kind {
		case "sent":
			snap.Sent = r.Cnt
		case "delivered":
			snap.Delivered = r.Cnt
		case "open":
			snap.Opened = r.Cnt
			snap.UniqueOpened = r.Uniq
		case "machine_open":
			snap.MachineOpen = r.Cnt
		case "click":
			snap.Clicked = r.Cnt
			snap.UniqueClick = r.Uniq
		case "unsub":
			snap.Unsubscribed = r.Cnt
		case "complaint":
			snap.Complained = r.Cnt
		case "bounce":
			snap.Bounced = r.Cnt
		case "skip":
			snap.Skipped = r.Cnt
		case "fail":
			snap.Failed = r.Cnt
		}
	}
	var queued int64
	s.DB.Model(&model.CampaignRecipient{}).Where("tenant_id = ? AND campaign_id = ?", tenant, campaign).Count(&queued)
	snap.Queued = queued
	return snap, nil
}

func (s Store) Reputation(tenant, campaign string) (hardRate, complaintRate float64, sent int64, err error) {
	s.DB.Model(&model.EmailEvent{}).Where("tenant_id = ? AND campaign_id = ? AND kind = ?", tenant, campaign, "sent").Count(&sent)
	if sent == 0 {
		return 0, 0, 0, nil
	}
	var hard, comp int64
	s.DB.Model(&model.EmailEvent{}).Where("tenant_id = ? AND campaign_id = ? AND kind = ?", tenant, campaign, "bounce").Count(&hard)
	s.DB.Model(&model.EmailEvent{}).Where("tenant_id = ? AND campaign_id = ? AND kind = ?", tenant, campaign, "complaint").Count(&comp)
	return float64(hard) / float64(sent), float64(comp) / float64(sent), sent, nil
}

func (s Store) DueScheduled(now time.Time) ([]model.Campaign, error) {
	var rows []model.Campaign
	err := s.DB.Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", "scheduled", now).Find(&rows).Error
	return rows, err
}

func (s Store) RunningCampaigns() ([]model.Campaign, error) {
	var rows []model.Campaign
	err := s.DB.Where("status = ?", "running").Find(&rows).Error
	return rows, err
}

func JSONMap(m map[string]string) datatypes.JSON {
	if m == nil {
		return datatypes.JSON([]byte("{}"))
	}
	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
}

func mapErr(err error) error {
	if err == gorm.ErrRecordNotFound {
		return domain.ErrNotFound
	}
	return err
}
