package service_test

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/lumen/relay/internal/repo"
	"github.com/lumen/relay/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImporterKeepsEachCSVRecordIndependent(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=test dbname=test sslmode=disable",
		PreferSimpleProtocol: true,
		WithoutReturning:     true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("create dry-run store: %v", err)
	}

	longName := strings.Repeat("B", 129)
	data := []byte("email,name\nnot-an-email,First\nvalid@example.com," + longName + "\n")
	peek := data[:8]
	result, err := (service.Importer{Store: repo.Store{DB: db}}).Import(
		"tenant-1", "", "contacts.csv", bytes.NewReader(data[8:]), peek,
	)
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if result.Job.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Job.Total)
	}
	if result.Job.Failed != 2 {
		t.Fatalf("failed = %d, want 2", result.Job.Failed)
	}

	records, err := csv.NewReader(strings.NewReader(result.ErrorCSV)).ReadAll()
	if err != nil {
		t.Fatalf("parse error_csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("error_csv records = %d, want header plus two failed rows: %q", len(records), result.ErrorCSV)
	}
	if got := records[1][0]; got != "not-an-email" {
		t.Fatalf("failed email = %q, want not-an-email", got)
	}
	if got := records[2][0]; got != "valid@example.com" {
		t.Fatalf("second failed email = %q, want valid@example.com", got)
	}
}
