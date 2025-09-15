package porkbun

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, `{"status":"SUCCESS","yourIp":"127.0.0.1"}`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)

	resp, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if resp.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", resp.Status)
	}
}

func TestRetrieveRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, `{"status":"SUCCESS","records":[{"id":"12345","name":"example.com","type":"TXT","content":"v=spf1...","ttl":"300","prio":"0","notes":""}]}`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)

	resp, err := client.RetrieveRecords("example.com")
	if err != nil {
		t.Fatalf("RetrieveRecords failed: %v", err)
	}

	if resp.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", resp.Status)
	}

	if len(resp.Records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(resp.Records))
	}

	if resp.Records[0].ID != "12345" {
		t.Errorf("Expected record ID 12345, got %s", resp.Records[0].ID)
	}
}

func TestUpdateRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, `{"status":"SUCCESS"}`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)

	resp, err := client.UpdateRecord("example.com", "12345", "v=spf1 ...")
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}

	if resp.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", resp.Status)
	}
}

func TestPing_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		if _, err := fmt.Fprintln(w, `Internal Server Error`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)
	_, err := client.Ping()
	if err == nil {
		t.Errorf("Expected error for server failure, got nil")
	}
}

func TestPing_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, `not a json response`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)
	_, err := client.Ping()
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got nil")
	}
}

func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		if _, err := fmt.Fprintln(w, `{"status":"SUCCESS","yourIp":"127.0.0.1"}`); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithURL("api_key", "secret_key", server.URL, false)
	client.SetTimeout(1 * time.Millisecond)
	_, err := client.Ping()
	if err == nil {
		t.Errorf("Expected timeout error, got nil")
	}
}
