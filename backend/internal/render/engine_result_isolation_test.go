package render_test

import (
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/render"
)

func TestEngineResultsKeepIndependentWarnings(t *testing.T) {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	first, err := render.Engine(domain.TemplateAST{Blocks: []domain.TemplateBlock{
		{Type: "text", HTML: "FIRST-BODY"},
		{Type: "legacy"},
	}}, render.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first.Warnings, []string{"unknown_block:legacy"}) {
		t.Fatalf("first warnings = %v", first.Warnings)
	}

	second, err := render.Engine(domain.TemplateAST{Blocks: []domain.TemplateBlock{
		{Type: "text", HTML: "SECOND-BODY"},
		{Type: "video"},
	}}, render.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(second.Warnings, []string{"unknown_block:video"}) {
		t.Fatalf("second warnings = %v", second.Warnings)
	}
	if !slices.Equal(first.Warnings, []string{"unknown_block:legacy"}) {
		t.Fatalf("first result changed after rendering another template: %v", first.Warnings)
	}
	if !strings.Contains(first.HTML, "FIRST-BODY") || strings.Contains(first.HTML, "SECOND-BODY") {
		t.Fatalf("first HTML changed after rendering another template: %s", first.HTML)
	}
}
