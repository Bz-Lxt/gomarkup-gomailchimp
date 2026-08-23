package domain

import "time"

type Tenant struct {
	ID           string
	Name         string
	DailyQuota   int
	MonthlyQuota int
	CreatedAt    time.Time
}

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	Name         string
	Role         string // owner | marketer | viewer
}

type Claims struct {
	UserID   string
	TenantID string
	Email    string
	Role     string
}

func (c Claims) CanWrite() bool {
	return c.Role == "owner" || c.Role == "marketer"
}

func (c Claims) IsOwner() bool {
	return c.Role == "owner"
}

type Contact struct {
	ID         string
	TenantID   string
	Email      string
	Name       string
	Attrs      map[string]string
	Status     string
	SoftBounce int
}

type TemplateAST struct {
	Width      int            `json:"width"`
	Background string         `json:"background"`
	Blocks     []TemplateBlock `json:"blocks"`
}

type TemplateBlock struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // text | image | button | divider
	HTML     string `json:"html,omitempty"`
	Align    string `json:"align,omitempty"`
	Color    string `json:"color,omitempty"`
	FontSize int    `json:"fontSize,omitempty"`
	Src      string `json:"src,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Label    string `json:"label,omitempty"`
	URL      string `json:"url,omitempty"`
	Bg       string `json:"bg,omitempty"`
	Padding  int    `json:"padding,omitempty"`
}

type SendJob struct {
	RecipientID string `json:"recipient_id"`
	CampaignID  string `json:"campaign_id"`
	TenantID    string `json:"tenant_id"`
	Attempt     int    `json:"attempt"`
	DomainHint  string `json:"domain_hint"`
	ChannelKey  string `json:"channel_key"`
}

type BounceClass string

const (
	BounceHard  BounceClass = "hard"
	BounceSoft  BounceClass = "soft"
	BounceBlock BounceClass = "block"
	BounceOK    BounceClass = "ok"
)

type BounceEvent struct {
	Email      string
	Class      BounceClass
	Code       string
	Enhanced   string
	Message    string
	CampaignID string
	TenantID   string
	Source     string
}

type FunnelSnapshot struct {
	CampaignID   string `json:"campaign_id"`
	Queued       int64  `json:"queued"`
	Sent         int64  `json:"sent"`
	Delivered    int64  `json:"delivered"`
	Opened       int64  `json:"opened"`
	UniqueOpened int64  `json:"unique_opened"`
	MachineOpen  int64  `json:"machine_open"`
	Clicked      int64  `json:"clicked"`
	UniqueClick  int64  `json:"unique_click"`
	Unsubscribed int64  `json:"unsubscribed"`
	Complained   int64  `json:"complained"`
	Bounced      int64  `json:"bounced"`
	Skipped      int64  `json:"skipped"`
	Failed       int64  `json:"failed"`
}
