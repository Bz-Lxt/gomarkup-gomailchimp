package render

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/token"
)

var hrefRe = regexp.MustCompile(`(?i)href="(https?://[^"]+)"`)

var warningBuffers = sync.Pool{
	New: func() any { return make([]string, 0, 8) },
}

type Result struct {
	HTML     string
	Text     string
	Warnings []string
}

type Context struct {
	Vars          map[string]string
	Secret        string
	TrackBase     string
	UnsubBase     string
	TenantID      string
	CampaignID    string
	RecipientID   string
	IncludePixel  bool
	RewriteLinks  bool
}

func Engine(ast domain.TemplateAST, ctx Context) (Result, error) {
	if ast.Width <= 0 {
		ast.Width = 600
	}
	if ast.Background == "" {
		ast.Background = "#f4efe6"
	}
	var body strings.Builder
	warn := warningBuffers.Get().([]string)[:0]
	defer func() { warningBuffers.Put(warn[:0]) }()
	for _, b := range ast.Blocks {
		htmlBlock, w := renderBlock(b)
		body.WriteString(htmlBlock)
		warn = append(warn, w...)
	}
	inner := body.String()
	rendered, missing := applyVars(inner, ctx.Vars)
	warn = append(warn, missing...)

	if ctx.RewriteLinks && ctx.Secret != "" {
		rendered = rewriteLinks(rendered, ctx)
	}

	unsubURL := ctx.UnsubBase + "/u?t=" + url.QueryEscape(token.UnsubToken(ctx.Secret, ctx.TenantID, ctx.CampaignID, ctx.RecipientID))
	pixel := ""
	if ctx.IncludePixel && ctx.Secret != "" {
		pixel = fmt.Sprintf(`<img src="%s%s" width="1" height="1" alt="" style="display:block;width:1px;height:1px;border:0;" />`,
			ctx.TrackBase, token.OpenPath(ctx.Secret, ctx.TenantID, ctx.CampaignID, ctx.RecipientID))
	}

	doc := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>email</title></head>
<body style="margin:0;padding:0;background:%s;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;"><tr><td align="center">
<table role="presentation" width="%d" cellpadding="0" cellspacing="0" style="width:%dpx;max-width:100%%;background:#fff;">
%s
<tr><td style="padding:20px;font-family:Georgia,serif;font-size:12px;color:#8a8175;text-align:center;">
若不愿再收到此类邮件，请<a href="%s" style="color:#8a8175;">点此退订</a>
</td></tr>
</table>%s
</td></tr></table>
</body></html>`, ast.Background, ast.Background, ast.Width, ast.Width, rendered, html.EscapeString(unsubURL), pixel)

	// unsub URL must not be escaped as text — rebuild last link without EscapeString on href
	doc = strings.Replace(doc, html.EscapeString(unsubURL), unsubURL, 1)

	plain := stripTags(rendered) + "\n退订: " + unsubURL
	return Result{HTML: doc, Text: plain, Warnings: warn}, nil
}

func renderBlock(b domain.TemplateBlock) (string, []string) {
	align := b.Align
	if align == "" {
		align = "left"
	}
	color := b.Color
	if color == "" {
		color = "#2b2118"
	}
	fs := b.FontSize
	if fs == 0 {
		fs = 16
	}
	pad := b.Padding
	if pad == 0 {
		pad = 16
	}
	switch b.Type {
	case "text":
		return fmt.Sprintf(`<tr><td style="padding:%dpx;font-family:Georgia,'Times New Roman',serif;font-size:%dpx;line-height:1.6;color:%s;text-align:%s;">%s</td></tr>`,
			pad, fs, color, align, b.HTML), nil
	case "image":
		alt := html.EscapeString(b.Alt)
		src := html.EscapeString(b.Src)
		return fmt.Sprintf(`<tr><td style="padding:%dpx;text-align:%s;"><img src="%s" alt="%s" style="max-width:100%%;height:auto;border:0;display:block;" /></td></tr>`,
			pad, align, src, alt), nil
	case "button":
		bg := b.Bg
		if bg == "" {
			bg = "#c45c26"
		}
		label := html.EscapeString(b.Label)
		href := html.EscapeString(b.URL)
		return fmt.Sprintf(`<tr><td style="padding:%dpx;text-align:%s;">
<a href="%s" style="display:inline-block;background:%s;color:#fff8ef;font-family:Helvetica,Arial,sans-serif;font-size:14px;text-decoration:none;padding:12px 28px;border-radius:999px;">%s</a>
</td></tr>`, pad, align, href, bg, label), nil
	case "divider":
		c := b.Color
		if c == "" {
			c = "#e6dccb"
		}
		return fmt.Sprintf(`<tr><td style="padding:%dpx;"><hr style="border:none;border-top:1px solid %s;margin:0;" /></td></tr>`, pad, c), nil
	default:
		return "", []string{"unknown_block:" + b.Type}
	}
}

func applyVars(src string, vars map[string]string) (string, []string) {
	missing := []string{}
	tpl, err := template.New("mail").Option("missingkey=zero").Parse(src)
	if err != nil {
		return src, []string{"template_parse:" + err.Error()}
	}
	data := map[string]string{}
	for k, v := range vars {
		data[k] = v
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return src, []string{"template_exec:" + err.Error()}
	}
	// detect leftover {{ .X }}
	if strings.Contains(buf.String(), "{{") {
		missing = append(missing, "unresolved_placeholder")
	}
	return buf.String(), missing
}

func rewriteLinks(htmlDoc string, ctx Context) string {
	return hrefRe.ReplaceAllStringFunc(htmlDoc, func(m string) string {
		sub := hrefRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		dest := sub[1]
		if strings.Contains(dest, "/u?t=") || strings.Contains(dest, "/c/") {
			return m
		}
		path := token.ClickPath(ctx.Secret, ctx.TenantID, ctx.CampaignID, ctx.RecipientID, dest)
		return `href="` + ctx.TrackBase + path + `"`
	})
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return strings.TrimSpace(tagRe.ReplaceAllString(s, " "))
}
