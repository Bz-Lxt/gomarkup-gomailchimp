package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/model"
	"github.com/lumen/relay/internal/repo"
	"github.com/xuri/excelize/v2"
)

type Importer struct{ Store repo.Store }

type ImportResult struct {
	Job      model.ImportJob `json:"job"`
	ErrorCSV string          `json:"error_csv,omitempty"`
}

var allowedHeaders = map[string]string{
	"email": "email", "邮箱": "email", "e-mail": "email", "mail": "email",
	"name": "name", "姓名": "name", "username": "name",
}

func (im Importer) Import(tenant, listID, filename string, r io.Reader, peek []byte) (ImportResult, error) {
	rows, err := readTable(peek, r)
	if err != nil {
		return ImportResult{}, err
	}
	if len(rows) < 2 {
		return ImportResult{}, fmt.Errorf("%w: empty table", domain.ErrValidation)
	}
	header := rows[0]
	idx := map[string]int{}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if canon, ok := allowedHeaders[key]; ok {
			idx[canon] = i
		} else if key != "" {
			idx["attr:"+h] = i
		}
	}
	if _, ok := idx["email"]; !ok {
		return ImportResult{}, fmt.Errorf("%w: missing email column", domain.ErrValidation)
	}

	job := model.ImportJob{
		ID: uuid.NewString(), TenantID: tenant, ListID: listID, Filename: filename,
		Status: "running", CreatedAt: clock.Now(),
	}
	_ = im.Store.DB.Create(&job).Error

	var errBuf bytes.Buffer
	ew := csv.NewWriter(&errBuf)
	_ = ew.Write(append(header, "error"))

	var memberIDs []string
	for _, row := range rows[1:] {
		job.Total++
		email := get(row, idx["email"])
		name := get(row, idx["name"])
		if err := validateRow(email, name); err != nil {
			job.Failed++
			_ = ew.Write(append(pad(row, len(header)), err.Error()))
			continue
		}
		attrs := map[string]string{}
		for k, i := range idx {
			if strings.HasPrefix(k, "attr:") {
				attrs[strings.TrimPrefix(k, "attr:")] = get(row, i)
			}
		}
		c := &model.Contact{
			TenantID: tenant, Email: email, Name: name, Status: "subscribed", Attrs: repo.JSONMap(attrs),
		}
		created, err := im.Store.UpsertContact(c)
		if err != nil {
			job.Failed++
			_ = ew.Write(append(pad(row, len(header)), err.Error()))
			continue
		}
		if created {
			job.Imported++
		} else {
			job.Updated++
		}
		found, _ := im.Store.FindContact(tenant, email)
		if found != nil {
			memberIDs = append(memberIDs, found.ID)
		}
	}
	if listID != "" && len(memberIDs) > 0 {
		_ = im.Store.AddMembers(tenant, listID, memberIDs)
	}
	ew.Flush()
	job.ErrorCSV = errBuf.String()
	job.Status = "done"
	_ = im.Store.DB.Save(&job).Error
	return ImportResult{Job: job, ErrorCSV: job.ErrorCSV}, nil
}

func validateRow(email, name string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email required")
	}
	if utf8.RuneCountInString(email) > 255 {
		return fmt.Errorf("email too long")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email")
	}
	if utf8.RuneCountInString(name) > 128 {
		return fmt.Errorf("name too long")
	}
	return nil
}

func readTable(peek []byte, r io.Reader) ([][]string, error) {
	body := io.MultiReader(bytes.NewReader(peek), r)
	if bytes.HasPrefix(peek, []byte{0x50, 0x4b}) { // zip / xlsx
		f, err := excelize.OpenReader(body)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("%w: no sheet", domain.ErrValidation)
		}
		return f.GetRows(sheets[0])
	}
	cr := csv.NewReader(body)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	header, err := cr.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows := [][]string{header}
	for {
		row, err := cr.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, append([]string(nil), row...))
	}
}

func get(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func pad(row []string, n int) []string {
	if len(row) >= n {
		return row
	}
	out := make([]string, n)
	copy(out, row)
	return out
}
