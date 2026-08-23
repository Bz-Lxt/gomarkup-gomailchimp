package render

import (
	"strings"
	"testing"

	"github.com/lumen/relay/internal/domain"
)

func TestEngineEscapesAndPixel(t *testing.T) {
	ast := domain.TemplateAST{Blocks: []domain.TemplateBlock{
		{Type: "text", HTML: "Hi, {{ .UserName }}"},
		{Type: "button", Label: "Go", URL: "https://shop.example/a"},
	}}
	out, err := Engine(ast, Context{
		Vars: map[string]string{"UserName": `<script>x</script>`},
		Secret: "s", TrackBase: "http://t", UnsubBase: "http://u",
		TenantID: "ten", CampaignID: "c", RecipientID: "r",
		IncludePixel: true, RewriteLinks: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.HTML, "<script>x</script>") {
		t.Fatal("unescaped")
	}
	if !strings.Contains(out.HTML, "/o/") || !strings.Contains(out.HTML, ".gif") {
		t.Fatal("pixel missing", out.HTML)
	}
	if !strings.Contains(out.HTML, "/c/") {
		t.Fatal("click rewrite missing")
	}
	if strings.Contains(out.HTML, "https://shop.example/a") {
		t.Fatal("raw dest leaked")
	}
}

func TestUnknownVarEmpty(t *testing.T) {
	ast := domain.TemplateAST{Blocks: []domain.TemplateBlock{{Type: "text", HTML: "Hi, {{ .Nope }}"}}}
	out, err := Engine(ast, Context{Vars: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.HTML, "{{") {
		t.Fatal(out.HTML)
	}
}
