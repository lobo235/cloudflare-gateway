package cloudflare_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cf "github.com/cloudflare/cloudflare-go"

	cfclient "github.com/lobo235/cloudflare-gateway/internal/cloudflare"
)

// newTestClientAndServer creates a mock Cloudflare API server and a Client pointed at it.
// The mux parameter lets tests register handlers for specific API paths.
func newTestClientAndServer(t *testing.T, mux *http.ServeMux) (*cfclient.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)

	// Use the cloudflare-go library's option to point at our test server.
	api, err := cf.NewWithAPIToken("test-token", cf.BaseURL(srv.URL+"/client/v4"))
	if err != nil {
		t.Fatalf("creating test cf api: %v", err)
	}
	client := cfclient.NewClientFromAPI(api)
	return client, srv
}

func TestPing_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []cf.Zone{})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPing_Failure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []map[string]any{{"code": 1000, "message": "server error"}},
			"result":  nil,
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestListZones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []map[string]any{
			{"id": "zone1", "name": "example.com", "status": "active"},
			{"id": "zone2", "name": "example.org", "status": "active"},
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("got %d zones, want 2", len(zones))
	}
	if zones[0].Name != "example.com" {
		t.Errorf("zones[0].Name = %q, want example.com", zones[0].Name)
	}
}

func TestListDNSRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []map[string]any{
			{"id": "rec1", "type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1},
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	records, err := client.ListDNSRecords(context.Background(), "zone1", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Type != "CNAME" {
		t.Errorf("Type = %q, want CNAME", records[0].Type)
	}
}

func TestGetDNSRecord(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, map[string]any{
			"id": "rec1", "type": "A", "name": "test.example.com", "content": "1.2.3.4", "ttl": 300,
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	record, err := client.GetDNSRecord(context.Background(), "zone1", "rec1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.ID != "rec1" {
		t.Errorf("ID = %q, want rec1", record.ID)
	}
	if record.Content != "1.2.3.4" {
		t.Errorf("Content = %q, want 1.2.3.4", record.Content)
	}
}

func TestCreateDNSRecord(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeCloudflareResponse(w, map[string]any{
			"id": "new-rec", "type": "CNAME", "name": "new.example.com", "content": "example.com", "ttl": 1,
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	rec := cfclient.DNSRecord{Type: "CNAME", Name: "new.example.com", Content: "example.com", TTL: 1}
	created, err := client.CreateDNSRecord(context.Background(), "zone1", rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "new-rec" {
		t.Errorf("ID = %q, want new-rec", created.ID)
	}
}

func TestUpdateDNSRecord(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeCloudflareResponse(w, map[string]any{
			"id": "rec1", "type": "A", "name": "test.example.com", "content": "5.6.7.8", "ttl": 300,
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	rec := cfclient.DNSRecord{Type: "A", Name: "test.example.com", Content: "5.6.7.8", TTL: 300}
	updated, err := client.UpdateDNSRecord(context.Background(), "zone1", "rec1", rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Content != "5.6.7.8" {
		t.Errorf("Content = %q, want 5.6.7.8", updated.Content)
	}
}

func TestNewClient_Success(t *testing.T) {
	client, err := cfclient.NewClient("test-api-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestListZones_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusInternalServerError, 1000, "internal error")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	_, err := client.ListZones(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetZoneIDByName_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []map[string]any{
			{"id": "zone1", "name": "example.com", "status": "active"},
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	id, err := client.GetZoneIDByName(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "zone1" {
		t.Errorf("zone ID = %q, want zone1", id)
	}
}

func TestGetZoneIDByName_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []map[string]any{})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	_, err := client.GetZoneIDByName(context.Background(), "nonexistent.com")
	if err == nil {
		t.Fatal("expected error for non-existent zone")
	}
}

func TestListDNSRecords_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusInternalServerError, 1000, "internal error")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	_, err := client.ListDNSRecords(context.Background(), "zone1", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListDNSRecords_WithFilters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareResponse(w, []map[string]any{
			{"id": "rec1", "type": "A", "name": "test.example.com", "content": "1.2.3.4", "ttl": 300},
		})
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	records, err := client.ListDNSRecords(context.Background(), "zone1", "A", "test.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
}

func TestGetDNSRecord_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusNotFound, 1001, "not found")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	_, err := client.GetDNSRecord(context.Background(), "zone1", "rec1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateDNSRecord_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusBadRequest, 1002, "invalid record")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	rec := cfclient.DNSRecord{Type: "CNAME", Name: "new.example.com", Content: "example.com", TTL: 1}
	_, err := client.CreateDNSRecord(context.Background(), "zone1", rec)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateDNSRecord_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusBadRequest, 1003, "invalid update")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	rec := cfclient.DNSRecord{Type: "A", Name: "test.example.com", Content: "5.6.7.8", TTL: 300}
	_, err := client.UpdateDNSRecord(context.Background(), "zone1", "rec1", rec)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteDNSRecord_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareErrorResponse(w, http.StatusInternalServerError, 1004, "delete failed")
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	err := client.DeleteDNSRecord(context.Background(), "zone1", "rec1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/zone1/dns_records/rec1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeCloudflareResponse(w, nil)
	})
	client, srv := newTestClientAndServer(t, mux)
	defer srv.Close()

	if err := client.DeleteDNSRecord(context.Background(), "zone1", "rec1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// writeCloudflareResponse writes a Cloudflare-style API response envelope.
func writeCloudflareResponse(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"success":  true,
		"errors":   []any{},
		"messages": []any{},
		"result":   result,
	}
	// Add result_info for list endpoints
	if _, ok := result.([]map[string]any); ok {
		resp["result_info"] = map[string]any{
			"page":        1,
			"per_page":    50,
			"total_pages": 1,
			"count":       len(result.([]map[string]any)),
			"total_count": len(result.([]map[string]any)),
		}
	}
	json.NewEncoder(w).Encode(resp)
}

// writeCloudflareErrorResponse writes a Cloudflare-style error response.
func writeCloudflareErrorResponse(w http.ResponseWriter, status, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success":  false,
		"errors":   []map[string]any{{"code": code, "message": message}},
		"messages": []any{},
		"result":   nil,
	})
}
