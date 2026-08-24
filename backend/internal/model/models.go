package model

import (
	"time"

	"github.com/lumen/relay/internal/clock"
	"gorm.io/datatypes"
)

func now() time.Time { return clock.Now() }

type Tenant struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	DailyQuota   int       `json:"daily_quota" gorm:"not null;default:50000"`
	MonthlyQuota int       `json:"monthly_quota" gorm:"not null;default:500000"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null"`
}

func (Tenant) TableName() string { return "tenants" }

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID     string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Email        string    `json:"email" gorm:"size:255;not null;uniqueIndex"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	Role         string    `json:"role" gorm:"size:32;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null"`
}

func (User) TableName() string { return "users" }

type Contact struct {
	ID         string         `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID   string         `json:"tenant_id" gorm:"type:uuid;uniqueIndex:uidx_contact_email;not null"`
	Email      string         `json:"email" gorm:"size:255;uniqueIndex:uidx_contact_email;not null"`
	Name       string         `json:"name" gorm:"size:128"`
	Attrs      datatypes.JSON `json:"attrs" gorm:"type:jsonb"`
	Status     string         `json:"status" gorm:"size:32;not null;default:subscribed"`
	SoftBounce int            `json:"soft_bounce" gorm:"not null;default:0"`
	CreatedAt  time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"not null"`
}

func (Contact) TableName() string { return "contacts" }

type ContactList struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Name      string    `json:"name" gorm:"size:128;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

func (ContactList) TableName() string { return "contact_lists" }

type ListMembership struct {
	ListID    string    `gorm:"primaryKey;type:uuid"`
	ContactID string    `gorm:"primaryKey;type:uuid"`
	TenantID  string    `gorm:"type:uuid;index;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (ListMembership) TableName() string { return "list_memberships" }

type Template struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Name      string    `json:"name" gorm:"size:128;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null"`
}

func (Template) TableName() string { return "templates" }

type TemplateVersion struct {
	ID         string         `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID   string         `json:"tenant_id" gorm:"type:uuid;index;not null"`
	TemplateID string         `json:"template_id" gorm:"type:uuid;index;not null"`
	Version    int            `json:"version" gorm:"not null"`
	Subject    string         `json:"subject" gorm:"size:255;not null"`
	AST        datatypes.JSON `json:"ast" gorm:"type:jsonb;not null"`
	CreatedAt  time.Time      `json:"created_at" gorm:"not null"`
}

func (TemplateVersion) TableName() string { return "template_versions" }

type Campaign struct {
	ID              string     `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID        string     `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Name            string     `json:"name" gorm:"size:128;not null"`
	Status          string     `json:"status" gorm:"size:32;not null;default:draft"`
	FromName        string     `json:"from_name" gorm:"size:128;not null"`
	FromEmail       string     `json:"from_email" gorm:"size:255;not null"`
	ReplyTo         string     `json:"reply_to" gorm:"size:255"`
	Subject         string     `json:"subject" gorm:"size:255;not null"`
	ListID          string     `json:"list_id" gorm:"type:uuid"`
	TemplateVerID   string     `json:"template_ver_id" gorm:"type:uuid"`
	Strategy        string     `json:"strategy" gorm:"size:32;not null;default:immediate"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	BatchSize       int        `json:"batch_size" gorm:"not null;default:200"`
	BatchIntervalS  int        `json:"batch_interval_s" gorm:"not null;default:60"`
	RampPercent     int        `json:"ramp_percent" gorm:"not null;default:20"`
	ChannelStrategy string     `json:"channel_strategy" gorm:"size:32;not null;default:balanced"`
	PausedReason    string     `json:"paused_reason" gorm:"size:255"`
	CreatedAt       time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null"`
}

func (Campaign) TableName() string { return "campaigns" }

type CampaignRecipient struct {
	ID          string     `gorm:"primaryKey;type:uuid"`
	TenantID    string     `gorm:"type:uuid;index;not null"`
	CampaignID  string     `gorm:"type:uuid;uniqueIndex:uidx_rcpt;not null"`
	ContactID   string     `gorm:"type:uuid;uniqueIndex:uidx_rcpt;not null"`
	Email       string     `gorm:"size:255;not null"`
	Domain      string     `gorm:"size:128;index;not null"`
	Status      string     `gorm:"size:32;not null;default:pending"`
	ChannelKey  string     `gorm:"size:64"`
	MessageID   string     `gorm:"size:128;uniqueIndex"`
	Attempt     int        `gorm:"not null;default:0"`
	LastError   string     `gorm:"size:512"`
	SentAt      *time.Time
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (CampaignRecipient) TableName() string { return "campaign_recipients" }

type SendChannel struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID   string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Key        string    `json:"key" gorm:"size:64;not null"`
	Name       string    `json:"name" gorm:"size:128;not null"`
	Provider   string    `json:"provider" gorm:"size:32;not null"`
	Weight     int       `json:"weight" gorm:"not null;default:1"`
	Health     float64   `json:"health" gorm:"not null;default:1"`
	State      string    `json:"state" gorm:"size:16;not null;default:closed"`
	FailStreak int       `json:"fail_streak" gorm:"not null;default:0"`
	Host       string    `json:"host" gorm:"size:255"`
	Port       int       `json:"port"`
	Username   string    `json:"username" gorm:"size:255"`
	Password   string    `json:"-" gorm:"size:255"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null"`
}

func (SendChannel) TableName() string { return "send_channels" }

type EmailEvent struct {
	ID          string    `gorm:"primaryKey;type:uuid"`
	TenantID    string    `gorm:"type:uuid;index:idx_evt_camp,priority:1;not null"`
	CampaignID  string    `gorm:"type:uuid;index:idx_evt_camp,priority:2;not null"`
	RecipientID string    `gorm:"type:uuid;index"`
	Kind        string    `gorm:"size:32;not null"` // sent|delivered|open|click|bounce|unsub|complaint|skip|fail|machine_open
	UniqueFlag  bool      `gorm:"not null;default:false"`
	URL         string    `gorm:"size:1024"`
	Meta        datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time `gorm:"index;not null"`
}

func (EmailEvent) TableName() string { return "email_events" }

type Suppression struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;uniqueIndex:uidx_supp;not null"`
	Email     string    `json:"email" gorm:"size:255;uniqueIndex:uidx_supp;not null"`
	Reason    string    `json:"reason" gorm:"size:64;not null"`
	Source    string    `json:"source" gorm:"size:64;not null"`
	Detail    string    `json:"detail" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

func (Suppression) TableName() string { return "suppressions" }

type BounceRecord struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	TenantID  string    `gorm:"type:uuid;index;not null"`
	Email     string    `gorm:"size:255;index;not null"`
	Class     string    `gorm:"size:16;not null"`
	Code      string    `gorm:"size:16"`
	Enhanced  string    `gorm:"size:32"`
	Message   string    `gorm:"size:512"`
	Source    string    `gorm:"size:32;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (BounceRecord) TableName() string { return "bounce_records" }

type ImportJob struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	ListID    string    `json:"list_id" gorm:"type:uuid"`
	Filename  string    `json:"filename" gorm:"size:255"`
	Status    string    `json:"status" gorm:"size:32;not null"`
	Total     int       `json:"total" gorm:"not null;default:0"`
	Imported  int       `json:"imported" gorm:"not null;default:0"`
	Updated   int       `json:"updated" gorm:"not null;default:0"`
	Failed    int       `json:"failed" gorm:"not null;default:0"`
	ErrorCSV  string    `json:"error_csv" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

func (ImportJob) TableName() string { return "import_jobs" }

type AuditLog struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	TenantID  string    `gorm:"type:uuid;index;not null"`
	ActorID   string    `gorm:"type:uuid"`
	Action    string    `gorm:"size:64;not null"`
	Target    string    `gorm:"size:255"`
	Detail    string    `gorm:"size:1024"`
	CreatedAt time.Time `gorm:"not null"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func TouchNew() time.Time { return now() }
