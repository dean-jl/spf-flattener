package backup

import (
	"time"
)

// Package backup provides DNS record backup and restore functionality.

// DNSRecord represents a single DNS record with all necessary fields for backup and restore.
type DNSRecord struct {
	ID        string    `json:"id" example:"rec_123456"`
	Name      string    `json:"name" example:"@"`
	Type      string    `json:"type" example:"A"`
	Content   string    `json:"content" example:"192.168.1.1"`
	TTL       int       `json:"ttl" example:"3600"`
	Priority  int       `json:"priority,omitempty" example:"10"`
	Notes     string    `json:"notes,omitempty" example:"Main website"`
	CreatedAt time.Time `json:"created_at,omitempty" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at,omitempty" example:"2024-01-01T00:00:00Z"`
}

// DNSRecordSet represents a collection of DNS records for a domain.
type DNSRecordSet struct {
	Domain      string      `json:"domain" example:"example.com"`
	Records     []DNSRecord `json:"records"`
	ExportedAt  time.Time   `json:"exported_at" example:"2024-01-15T10:30:00Z"`
	Provider    string      `json:"provider" example:"porkbun"`
	Version     string      `json:"version" example:"1.0"`
	Attribution string      `json:"attribution,omitempty" example:"Data retrieved via Porkbun API (https://porkbun.com/api)"`
}

// ExportOptions contains configuration options for exporting DNS records.
type ExportOptions struct {
	RecordTypes []string `json:"record_types" example:"[\"A\", \"AAAA\", \"CNAME\"]"` // Filter for specific record types (empty = all types)
	DryRun      bool     `json:"dry_run" example:"true"`
	Verbose     bool     `json:"verbose" example:"false"`
}

// ImportOptions contains configuration options for importing DNS records.
type ImportOptions struct {
	RecordTypes      []string `json:"record_types" example:"[\"A\", \"AAAA\"]"` // Filter for specific record types (empty = all types)
	DryRun           bool     `json:"dry_run" example:"true"`
	ReplaceExisting  bool     `json:"replace_existing" example:"false"`
	ConflictStrategy string   `json:"conflict_strategy" example:"skip" enum:"skip,replace,merge,abort"`
	BackupBefore     bool     `json:"backup_before" example:"true"`
	Verbose          bool     `json:"verbose" example:"false"`
}

// ImportResult contains the results of an import operation.
type ImportResult struct {
	Domain       string        `json:"domain"`
	TotalRecords int           `json:"total_records"`
	Created      int           `json:"created"`
	Updated      int           `json:"updated"`
	Skipped      int           `json:"skipped"`
	Failed       int           `json:"failed"`
	Conflicts    []Conflict    `json:"conflicts,omitempty"`
	Errors       []error       `json:"errors,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// Conflict represents a conflict between existing and imported records.
type Conflict struct {
	ExistingRecord DNSRecord `json:"existing_record"`
	ImportedRecord DNSRecord `json:"imported_record"`
	ConflictType   string    `json:"conflict_type" example:"name_type_mismatch"`
	Resolution     string    `json:"resolution,omitempty" example:"skipped"`
}
