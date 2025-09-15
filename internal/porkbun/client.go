package porkbun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var apiBaseURL = "https://api.porkbun.com/api/json/v3"

// BackupDNSRecord represents a DNS record for backup/restore operations.
// This is a simplified version of backup.DNSRecord to avoid circular imports.
type BackupDNSRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type Client struct {
	apiKey    string
	secretKey string
	client    *http.Client
	baseURL   string
	debug     bool
}

func NewClient(apiKey, secretKey string, debug bool) *Client {
	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: apiBaseURL,
		debug:   debug,
	}
}

func NewClientWithURL(apiKey, secretKey, baseURL string, debug bool) *Client {
	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		client:    &http.Client{},
		baseURL:   baseURL,
		debug:     debug,
	}
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

type PingResponse struct {
	Status string `json:"status"`
	YourIP string `json:"yourIp"`
}

func (c *Client) Ping() (*PingResponse, error) {
	body := map[string]string{
		"apikey":       c.apiKey,
		"secretapikey": c.secretKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/ping", c.baseURL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check HTTP status code for rate limiting
	if err := c.checkHTTPStatus(resp, respBody); err != nil {
		return nil, err
	}

	if c.debug {
		fmt.Printf("[DEBUG] Raw response body: %s\n", string(respBody))
	}

	var pingResponse PingResponse
	err = json.Unmarshal(respBody, &pingResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}
	if c.debug {
		fmt.Printf("[DEBUG] Parsed PingResponse: %+v\n", pingResponse)
	}

	return &pingResponse, nil
}

type RetrieveRecordsRequest struct {
	APIKey       string `json:"apikey"`
	SecretAPIKey string `json:"secretapikey"`
}

type RetrieveRecordsResponse struct {
	Status  string   `json:"status"`
	Records []Record `json:"records"`
}

type Record struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio"`
	Notes   string `json:"notes"`
}

func (c *Client) RetrieveRecords(domain string) (*RetrieveRecordsResponse, error) {
	body := RetrieveRecordsRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/retrieve/%s", c.baseURL, domain), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check HTTP status code for rate limiting
	if err := c.checkHTTPStatus(resp, respBody); err != nil {
		return nil, err
	}

	var retrieveRecordsResponse RetrieveRecordsResponse
	err = json.Unmarshal(respBody, &retrieveRecordsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if retrieveRecordsResponse.Status != "SUCCESS" {
		return nil, fmt.Errorf("Porkbun API error retrieving records: %s", retrieveRecordsResponse.Status)
	}

	return &retrieveRecordsResponse, nil
}

type UpdateRecordRequest struct {
	APIKey       string `json:"apikey"`
	SecretAPIKey string `json:"secretapikey"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	TTL          string `json:"ttl,omitempty"`
	Prio         string `json:"prio,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type UpdateRecordResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (c *Client) UpdateRecord(domain, recordID, content string) (*UpdateRecordResponse, error) {
	return c.UpdateRecordWithDetails(domain, recordID, "", "TXT", content, "600", "", "")
}

// UpdateRecordWithDetails provides full control over all update parameters
func (c *Client) UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes string) (*UpdateRecordResponse, error) {
	body := UpdateRecordRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
		Name:         name,
		Type:         recordType,
		Content:      content,
		TTL:          ttl,
		Prio:         prio,
		Notes:        notes,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/edit/%s/%s", c.baseURL, domain, recordID), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var updateRecordResponse UpdateRecordResponse
	err = json.Unmarshal(respBody, &updateRecordResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if updateRecordResponse.Status != "SUCCESS" {
		if updateRecordResponse.Message != "" {
			return nil, fmt.Errorf("Porkbun API error updating record: %s - %s", updateRecordResponse.Status, updateRecordResponse.Message)
		}
		return nil, fmt.Errorf("Porkbun API error updating record: %s", updateRecordResponse.Status)
	}

	return &updateRecordResponse, nil
}

type CreateRecordRequest struct {
	APIKey       string `json:"apikey"`
	SecretAPIKey string `json:"secretapikey"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	TTL          string `json:"ttl,omitempty"`
	Prio         string `json:"prio,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type CreateRecordResponse struct {
	Status  string `json:"status"`
	ID      int    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *Client) CreateRecord(domain, name, recordType, content string, ttl int) (*CreateRecordResponse, error) {
	var ttlStr string
	if ttl > 0 {
		ttlStr = fmt.Sprintf("%d", ttl)
	}

	body := CreateRecordRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
		Name:         name,
		Type:         recordType,
		Content:      content,
		TTL:          ttlStr,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/create/%s", c.baseURL, domain), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var createRecordResponse CreateRecordResponse
	err = json.Unmarshal(respBody, &createRecordResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if createRecordResponse.Status != "SUCCESS" {
		if createRecordResponse.Message != "" {
			return nil, fmt.Errorf("Porkbun API error creating record: %s - %s", createRecordResponse.Status, createRecordResponse.Message)
		}
		return nil, fmt.Errorf("Porkbun API error creating record: %s", createRecordResponse.Status)
	}

	return &createRecordResponse, nil
}

// CreateRecordWithOptions creates a DNS record with full control over all optional parameters
func (c *Client) CreateRecordWithOptions(domain, name, recordType, content string, ttl int, prio string, notes string) (*CreateRecordResponse, error) {
	var ttlStr string
	if ttl > 0 {
		ttlStr = fmt.Sprintf("%d", ttl)
	}

	body := CreateRecordRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
		Name:         name,
		Type:         recordType,
		Content:      content,
		TTL:          ttlStr,
		Prio:         prio,
		Notes:        notes,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/create/%s", c.baseURL, domain), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var createRecordResponse CreateRecordResponse
	err = json.Unmarshal(respBody, &createRecordResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if createRecordResponse.Status != "SUCCESS" {
		if createRecordResponse.Message != "" {
			return nil, fmt.Errorf("Porkbun API error creating record: %s - %s", createRecordResponse.Status, createRecordResponse.Message)
		}
		return nil, fmt.Errorf("Porkbun API error creating record: %s", createRecordResponse.Status)
	}

	return &createRecordResponse, nil
}

type DeleteRecordRequest struct {
	APIKey       string `json:"apikey"`
	SecretAPIKey string `json:"secretapikey"`
}

type DeleteRecordResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (c *Client) DeleteRecord(domain, recordID string) (*DeleteRecordResponse, error) {
	body := DeleteRecordRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/delete/%s/%s", c.baseURL, domain, recordID), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deleteRecordResponse DeleteRecordResponse
	err = json.Unmarshal(respBody, &deleteRecordResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if deleteRecordResponse.Status != "SUCCESS" {
		if deleteRecordResponse.Message != "" {
			return nil, fmt.Errorf("Porkbun API error deleting record: %s - %s", deleteRecordResponse.Status, deleteRecordResponse.Message)
		}
		return nil, fmt.Errorf("Porkbun API error deleting record: %s", deleteRecordResponse.Status)
	}

	return &deleteRecordResponse, nil
}

// DeleteRecordByNameType deletes DNS records by domain, subdomain, and type
// Uses the deleteByNameType endpoint: /dns/deleteByNameType/[DOMAIN]/[TYPE]/[SUBDOMAIN]
func (c *Client) DeleteRecordByNameType(domain, recordType, subdomain string) (*DeleteRecordResponse, error) {
	body := DeleteRecordRequest{
		APIKey:       c.apiKey,
		SecretAPIKey: c.secretKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/dns/deleteByNameType/%s/%s", c.baseURL, domain, recordType)
	if subdomain != "" {
		url = fmt.Sprintf("%s/%s", url, subdomain)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deleteRecordResponse DeleteRecordResponse
	err = json.Unmarshal(respBody, &deleteRecordResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, body: %s", err, string(respBody))
	}

	// Check for API-level error status
	if deleteRecordResponse.Status != "SUCCESS" {
		if deleteRecordResponse.Message != "" {
			return nil, fmt.Errorf("Porkbun API error deleting record by name/type: %s - %s", deleteRecordResponse.Status, deleteRecordResponse.Message)
		}
		return nil, fmt.Errorf("Porkbun API error deleting record by name/type: %s", deleteRecordResponse.Status)
	}

	return &deleteRecordResponse, nil
}

// RetrieveAllRecords retrieves all DNS records for a domain and converts them to BackupDNSRecord format.
func (c *Client) RetrieveAllRecords(domain string) ([]BackupDNSRecord, error) {
	response, err := c.RetrieveRecords(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve records: %w", err)
	}

	records := make([]BackupDNSRecord, len(response.Records))
	for i, record := range response.Records {
		ttl := 3600 // Default TTL
		if record.TTL != "" {
			if parsedTTL, err := strconv.Atoi(record.TTL); err == nil {
				ttl = parsedTTL
			}
		}

		priority := 0
		if record.Prio != "" {
			if parsedPrio, err := strconv.Atoi(record.Prio); err == nil {
				priority = parsedPrio
			}
		}

		records[i] = BackupDNSRecord{
			ID:       record.ID,
			Name:     record.Name,
			Type:     record.Type,
			Content:  record.Content,
			TTL:      ttl,
			Priority: priority,
			Notes:    record.Notes,
		}
	}

	return records, nil
}

// BulkCreateRecords creates multiple DNS records for a domain.
func (c *Client) BulkCreateRecords(domain string, records []BackupDNSRecord) error {
	for _, record := range records {
		var prio string
		if record.Priority > 0 {
			prio = strconv.Itoa(record.Priority)
		}

		_, err := c.CreateRecordWithOptions(domain, record.Name, record.Type, record.Content, record.TTL, prio, record.Notes)
		if err != nil {
			return fmt.Errorf("failed to create record %s %s: %w", record.Name, record.Type, err)
		}
	}
	return nil
}

// BulkUpdateRecords updates multiple DNS records for a domain.
func (c *Client) BulkUpdateRecords(domain string, records []BackupDNSRecord) error {
	for _, record := range records {
		var prio string
		if record.Priority > 0 {
			prio = strconv.Itoa(record.Priority)
		}
		var ttl string
		if record.TTL > 0 {
			ttl = strconv.Itoa(record.TTL)
		}

		_, err := c.UpdateRecordWithDetails(domain, record.ID, record.Name, record.Type, record.Content, ttl, prio, record.Notes)
		if err != nil {
			return fmt.Errorf("failed to update record %s %s: %w", record.Name, record.Type, err)
		}
	}
	return nil
}

// BulkDeleteRecords deletes multiple DNS records for a domain.
func (c *Client) BulkDeleteRecords(domain string, recordIDs []string) error {
	for _, recordID := range recordIDs {
		_, err := c.DeleteRecord(domain, recordID)
		if err != nil {
			return fmt.Errorf("failed to delete record %s: %w", recordID, err)
		}
	}
	return nil
}

// Attribution returns the attribution message required by Porkbun's terms of service
func (c *Client) Attribution() string {
	return "Data provided by Porkbun, LLC. Learn more at https://porkbun.com"
}

// DNSAPIClient abstracts DNS record management for extensibility.
type DNSAPIClient interface {
	Ping() (*PingResponse, error)
	RetrieveRecords(domain string) (*RetrieveRecordsResponse, error)
	UpdateRecord(domain, recordID, content string) (*UpdateRecordResponse, error)
	UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes string) (*UpdateRecordResponse, error)
	CreateRecord(domain, name, recordType, content string, ttl int) (*CreateRecordResponse, error)
	CreateRecordWithOptions(domain, name, recordType, content string, ttl int, prio string, notes string) (*CreateRecordResponse, error)
	DeleteRecord(domain, recordID string) (*DeleteRecordResponse, error)
	DeleteRecordByNameType(domain, recordType, subdomain string) (*DeleteRecordResponse, error)

	// Backup/Restore methods
	RetrieveAllRecords(domain string) ([]BackupDNSRecord, error)
	BulkCreateRecords(domain string, records []BackupDNSRecord) error
	BulkUpdateRecords(domain string, records []BackupDNSRecord) error
	BulkDeleteRecords(domain string, recordIDs []string) error

	// Attribution returns the attribution message for the DNS provider
	Attribution() string
}

// Ensure Porkbun Client implements DNSAPIClient
var _ DNSAPIClient = (*Client)(nil)

// checkHTTPStatus checks for HTTP rate limiting status codes and returns appropriate errors.
func (c *Client) checkHTTPStatus(resp *http.Response, respBody []byte) error {
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("HTTP %d %s: %s", resp.StatusCode, resp.Status, string(respBody))
	}
	return nil
}

func redactSensitive(input string) string {
	input = strings.ReplaceAll(input, "apikey", "[REDACTED]")
	input = strings.ReplaceAll(input, "secretapikey", "[REDACTED]")
	return input
}

// Example usage in error output:
// errMsg := redactSensitive(err.Error())
// logEvent(logger, "error", "API error", map[string]string{"error": errMsg})
