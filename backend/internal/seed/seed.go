package seed

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/lumen/relay/internal/auth"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/model"
	"gorm.io/gorm"
)

const TenantID = "11111111-1111-1111-1111-111111111111"
const ListID = "22222222-2222-2222-2222-222222222222"
const TplID = "33333333-3333-3333-3333-333333333333"
const VerID = "44444444-4444-4444-4444-444444444444"

func Run(db *gorm.DB) error {
	var n int64
	db.Model(&model.Tenant{}).Count(&n)
	if n > 0 {
		return nil
	}
	now := clock.Now()
	t := model.Tenant{ID: TenantID, Name: "北极星独立站", DailyQuota: 50000, MonthlyQuota: 500000, CreatedAt: now}
	if err := db.Create(&t).Error; err != nil {
		return err
	}
	users := []struct{ email, pw, name, role string }{
		{"owner@lumen.local", "Owner123!", "林深", "owner"},
		{"marketer@lumen.local", "Market123!", "苏晚", "marketer"},
		{"viewer@lumen.local", "Viewer123!", "何观", "viewer"},
	}
	for _, u := range users {
		h, _ := auth.HashPassword(u.pw)
		if err := db.Create(&model.User{
			ID: uuid.NewString(), TenantID: TenantID, Email: u.email, PasswordHash: h,
			Name: u.name, Role: u.role, CreatedAt: now,
		}).Error; err != nil {
			return err
		}
	}
	if err := db.Create(&model.ContactList{ID: ListID, TenantID: TenantID, Name: "核心唤醒客群", CreatedAt: now}).Error; err != nil {
		return err
	}
	contacts := []struct{ email, name string }{
		{"ada@gmail.com", "Ada"},
		{"ben@outlook.com", "Ben"},
		{"cara@example.com", "Cara"},
		{"dead@bounce.test", "Dead"},
	}
	for _, ct := range contacts {
		id := uuid.NewString()
		row := model.Contact{
			ID: id, TenantID: TenantID, Email: ct.email, Name: ct.name, Status: "subscribed",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		_ = db.Create(&model.ListMembership{ListID: ListID, ContactID: id, TenantID: TenantID, CreatedAt: now}).Error
	}
	ast := domain.TemplateAST{
		Width: 600, Background: "#f4efe6",
		Blocks: []domain.TemplateBlock{
			{ID: "h1", Type: "text", HTML: "Hi, {{ .UserName }}", Align: "left", Color: "#2b2118", FontSize: 22, Padding: 24},
			{ID: "p1", Type: "text", HTML: "你已经 21 天没有打开北极星了。我们为你留了一盏灯。", Align: "left", Color: "#5c5346", FontSize: 16, Padding: 16},
			{ID: "img", Type: "image", Src: "https://picsum.photos/seed/lumen/560/240", Alt: "暖光书桌", Align: "center"},
			{ID: "btn", Type: "button", Label: "回到书桌", URL: "https://example.com/back", Bg: "#c45c26", Align: "center"},
			{ID: "div", Type: "divider", Color: "#e6dccb"},
		},
	}
	raw, _ := json.Marshal(ast)
	if err := db.Create(&model.Template{ID: TplID, TenantID: TenantID, Name: "21 日唤醒", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		return err
	}
	if err := db.Create(&model.TemplateVersion{
		ID: VerID, TenantID: TenantID, TemplateID: TplID, Version: 1,
		Subject: "Hi, {{ .UserName }}，灯还亮着", AST: raw, CreatedAt: now,
	}).Error; err != nil {
		return err
	}
	chs := []model.SendChannel{
		{ID: uuid.NewString(), TenantID: TenantID, Key: "gmail-warmup", Name: "Gmail 预热通道", Provider: "smtp", Weight: 1, Health: 1, State: "closed", Host: "mailpit", Port: 1025, CreatedAt: now},
		{ID: uuid.NewString(), TenantID: TenantID, Key: "outlook-prod", Name: "Outlook 全速通道", Provider: "smtp", Weight: 2, Health: 1, State: "closed", Host: "mailpit", Port: 1025, CreatedAt: now},
	}
	for i := range chs {
		if err := db.Create(&chs[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
