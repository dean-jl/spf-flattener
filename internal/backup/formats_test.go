package backup

import (
	"testing"
	"time"
)

func TestJSONFormatHandler_Serialize(t *testing.T) {
	handler := &JSONFormatHandler{}

	recordSet := &DNSRecordSet{
		Domain:     "example.com",
		Provider:   "porkbun",
		Version:    "1.0",
		ExportedAt: time.Now(),
		Records: []DNSRecord{
			{
				ID:      "rec_123",
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
			},
		},
	}

	data, err := handler.Serialize(recordSet)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data is empty")
	}

	// Verify it contains expected content
	expectedContent := `"domain": "example.com"`
	if string(data) != expectedContent && !contains(string(data), expectedContent) {
		t.Errorf("Serialized data doesn't contain expected content: %s", expectedContent)
	}
}

func TestJSONFormatHandler_Deserialize(t *testing.T) {
	handler := &JSONFormatHandler{}

	jsonData := `{
		"domain": "example.com",
		"provider": "porkbun",
		"version": "1.0",
		"exported_at": "2024-01-15T10:30:00Z",
		"records": [
			{
				"id": "rec_123",
				"name": "@",
				"type": "A",
				"content": "192.168.1.1",
				"ttl": 3600
			}
		]
	}`

	recordSet, err := handler.Deserialize([]byte(jsonData))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if recordSet.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", recordSet.Domain)
	}
	if len(recordSet.Records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(recordSet.Records))
	}
	if recordSet.Records[0].Name != "@" {
		t.Errorf("Expected record name '@', got '%s'", recordSet.Records[0].Name)
	}
}

func TestTextFormatHandler_Serialize(t *testing.T) {
	handler := &TextFormatHandler{}

	recordSet := &DNSRecordSet{
		Domain:     "example.com",
		Provider:   "porkbun",
		Version:    "1.0",
		ExportedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Records: []DNSRecord{
			{
				ID:      "rec_123",
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
				Notes:   "Main website",
			},
			{
				ID:       "rec_456",
				Name:     "@",
				Type:     "MX",
				Content:  "mail.example.com",
				TTL:      3600,
				Priority: 10,
			},
		},
	}

	data, err := handler.Serialize(recordSet)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	content := string(data)
	if !contains(content, "DNS Records for example.com") {
		t.Error("Serialized text doesn't contain domain header")
	}
	if !contains(content, "A Records:") {
		t.Error("Serialized text doesn't contain A Records section")
	}
	if !contains(content, "MX Records:") {
		t.Error("Serialized text doesn't contain MX Records section")
	}
	if !contains(content, "@ → 192.168.1.1") {
		t.Error("Serialized text doesn't contain A record")
	}
	if !contains(content, "@ → mail.example.com") {
		t.Error("Serialized text doesn't contain MX record")
	}
}

func TestTextFormatHandler_Deserialize(t *testing.T) {
	handler := &TextFormatHandler{}

	textData := `DNS Records for example.com (Exported: 2024-01-15 10:30:00)
Provider: porkbun
Version: 1.0

A Records:
  @ → 192.168.1.1 (TTL: 3600, ID: rec_123)
    Notes: Main website

MX Records:
  @ → mail.example.com (Priority: 10, TTL: 3600, ID: rec_456)

`

	recordSet, err := handler.Deserialize([]byte(textData))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if recordSet.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", recordSet.Domain)
	}
	if recordSet.Provider != "porkbun" {
		t.Errorf("Expected provider 'porkbun', got '%s'", recordSet.Provider)
	}
	if len(recordSet.Records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(recordSet.Records))
	}
}

func TestGetFormatHandler(t *testing.T) {
	tests := []struct {
		format    string
		expectErr bool
		handler   FormatHandler
	}{
		{"json", false, &JSONFormatHandler{}},
		{"JSON", false, &JSONFormatHandler{}},
		{"txt", false, &TextFormatHandler{}},
		{"text", false, &TextFormatHandler{}},
		{"TEXT", false, &TextFormatHandler{}},
		{"xml", true, nil},
		{"", true, nil},
	}

	for _, test := range tests {
		handler, err := GetFormatHandler(test.format)
		if test.expectErr {
			if err == nil {
				t.Errorf("GetFormatHandler(%q) expected error, got nil", test.format)
			}
		} else {
			if err != nil {
				t.Errorf("GetFormatHandler(%q) unexpected error: %v", test.format, err)
			}
			if handler == nil {
				t.Errorf("GetFormatHandler(%q) returned nil handler", test.format)
			}
		}
	}
}

func TestGetFormatHandlerFromFilename(t *testing.T) {
	tests := []struct {
		filename  string
		expectErr bool
		handler   FormatHandler
	}{
		{"records.json", false, &JSONFormatHandler{}},
		{"backup.JSON", false, &JSONFormatHandler{}},
		{"output.txt", false, &TextFormatHandler{}},
		{"data.TXT", false, &TextFormatHandler{}},
		{"config.xml", true, nil},
		{"noextension", true, nil},
	}

	for _, test := range tests {
		handler, err := GetFormatHandlerFromFilename(test.filename)
		if test.expectErr {
			if err == nil {
				t.Errorf("GetFormatHandlerFromFilename(%q) expected error, got nil", test.filename)
			}
		} else {
			if err != nil {
				t.Errorf("GetFormatHandlerFromFilename(%q) unexpected error: %v", test.filename, err)
			}
			if handler == nil {
				t.Errorf("GetFormatHandlerFromFilename(%q) returned nil handler", test.filename)
			}
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(s)] != s[:len(s)-len(substr)] &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()
}
