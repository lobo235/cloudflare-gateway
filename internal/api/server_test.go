package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lobo235/cloudflare-gateway/internal/api"
	"github.com/lobo235/cloudflare-gateway/internal/cloudflare"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testAPIKey = "test-api-key"
const testVersion = "v1.0.0-test"

// mockCloudflare is a configurable mock that satisfies the cloudflareClient interface.
type mockCloudflare struct {
	pingFunc            func(ctx context.Context) error
	listZonesFunc       func(ctx context.Context) ([]cloudflare.Zone, error)
	getZoneIDByNameFunc func(ctx context.Context, name string) (string, error)
	listDNSRecordsFunc  func(ctx context.Context, zoneID, recordType, recordName string) ([]cloudflare.DNSRecord, error)
	getDNSRecordFunc    func(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error)
	createDNSRecordFunc func(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error)
	updateDNSRecordFunc func(ctx context.Context, zoneID, recordID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error)
	deleteDNSRecordFunc func(ctx context.Context, zoneID, recordID string) error
}

func (m *mockCloudflare) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockCloudflare) ListZones(ctx context.Context) ([]cloudflare.Zone, error) {
	if m.listZonesFunc != nil {
		return m.listZonesFunc(ctx)
	}
	return nil, nil
}

func (m *mockCloudflare) GetZoneIDByName(ctx context.Context, name string) (string, error) {
	if m.getZoneIDByNameFunc != nil {
		return m.getZoneIDByNameFunc(ctx, name)
	}
	return "", nil
}

func (m *mockCloudflare) ListDNSRecords(ctx context.Context, zoneID, recordType, recordName string) ([]cloudflare.DNSRecord, error) {
	if m.listDNSRecordsFunc != nil {
		return m.listDNSRecordsFunc(ctx, zoneID, recordType, recordName)
	}
	return nil, nil
}

func (m *mockCloudflare) GetDNSRecord(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error) {
	if m.getDNSRecordFunc != nil {
		return m.getDNSRecordFunc(ctx, zoneID, recordID)
	}
	return cloudflare.DNSRecord{}, nil
}

func (m *mockCloudflare) CreateDNSRecord(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
	if m.createDNSRecordFunc != nil {
		return m.createDNSRecordFunc(ctx, zoneID, rec)
	}
	return cloudflare.DNSRecord{}, nil
}

func (m *mockCloudflare) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
	if m.updateDNSRecordFunc != nil {
		return m.updateDNSRecordFunc(ctx, zoneID, recordID, rec)
	}
	return cloudflare.DNSRecord{}, nil
}

func (m *mockCloudflare) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	if m.deleteDNSRecordFunc != nil {
		return m.deleteDNSRecordFunc(ctx, zoneID, recordID)
	}
	return nil
}

// newTestServer creates a test HTTP server with the given mock cloudflare client.
func newTestServer(t *testing.T, mock *mockCloudflare) *httptest.Server {
	t.Helper()
	srv := api.NewServer(mock, testAPIKey, testVersion, discardLogger())
	return httptest.NewServer(srv.Handler())
}

func authHeader() string {
	return "Bearer " + testAPIKey
}

// --- helpers ---

func getJSON(t *testing.T, srv *httptest.Server, path string, auth bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if auth {
		req.Header.Set("Authorization", authHeader())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func putJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", path, err)
	}
	return resp
}

func deleteReq(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+path, nil)
	req.Header.Set("Authorization", authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

func assertErrorCode(t *testing.T, resp *http.Response, wantCode string) {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Code != wantCode {
		t.Errorf("error code = %q, want %q", body.Code, wantCode)
	}
}

// --- auth middleware ---

func TestAuth_MissingToken(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones", false)
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, resp, "unauthorized")
}

func TestAuth_WrongToken(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/zones", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_ValidToken(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		listZonesFunc: func(ctx context.Context) ([]cloudflare.Zone, error) {
			return []cloudflare.Zone{}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones", true)
	assertStatus(t, resp, http.StatusOK)
}

// --- GET /health ---

func TestHealth_CloudflareUp(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		pingFunc: func(ctx context.Context) error { return nil },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false)
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != testVersion {
		t.Errorf("version = %q, want %q", body.Version, testVersion)
	}
}

func TestHealth_CloudflareDown(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		pingFunc: func(ctx context.Context) error { return errors.New("connection refused") },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false)
	assertStatus(t, resp, http.StatusServiceUnavailable)

	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "unavailable" {
		t.Errorf("status = %q, want unavailable", body.Status)
	}
}

func TestHealth_NoAuthRequired(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	resp := getJSON(t, srv, "/health", false)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("/health should not require auth")
	}
}

// --- GET /zones ---

func TestListZones_OK(t *testing.T) {
	want := []cloudflare.Zone{
		{ID: "z1", Name: "example.com", Status: "active"},
		{ID: "z2", Name: "example.org", Status: "active"},
	}
	srv := newTestServer(t, &mockCloudflare{
		listZonesFunc: func(ctx context.Context) ([]cloudflare.Zone, error) { return want, nil },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/zones", true)
	assertStatus(t, resp, http.StatusOK)

	var got []cloudflare.Zone
	json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 2 {
		t.Fatalf("got %d zones, want 2", len(got))
	}
}

func TestListZones_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		listZonesFunc: func(ctx context.Context) ([]cloudflare.Zone, error) {
			return nil, errors.New("cloudflare unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /zones/{zoneID}/records ---

func TestListRecords_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return nil, errors.New("cloudflare unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones/z1/records", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestListRecords_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			if zoneID != "z1" {
				t.Errorf("zoneID = %q, want z1", zoneID)
			}
			return []cloudflare.DNSRecord{{ID: "r1", Type: "CNAME", Name: "test.example.com"}}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones/z1/records", true)
	assertStatus(t, resp, http.StatusOK)
}

func TestListRecords_WithFilters(t *testing.T) {
	var gotType, gotName string
	srv := newTestServer(t, &mockCloudflare{
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			gotType, gotName = rt, rn
			return []cloudflare.DNSRecord{}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones/z1/records?type=CNAME&name=test.example.com", true)
	assertStatus(t, resp, http.StatusOK)
	if gotType != "CNAME" {
		t.Errorf("type filter = %q, want CNAME", gotType)
	}
	if gotName != "test.example.com" {
		t.Errorf("name filter = %q, want test.example.com", gotName)
	}
}

// --- POST /zones/{zoneID}/records ---

func TestCreateRecord_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		createDNSRecordFunc: func(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			rec.ID = "new-id"
			return rec, nil
		},
	})
	defer srv.Close()

	resp := postJSON(t, srv, "/zones/z1/records", map[string]any{
		"type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1,
	})
	assertStatus(t, resp, http.StatusCreated)

	var got cloudflare.DNSRecord
	json.NewDecoder(resp.Body).Decode(&got)
	if got.ID != "new-id" {
		t.Errorf("ID = %q, want new-id", got.ID)
	}
}

func TestCreateRecord_MissingFields(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones/z1/records", map[string]any{"type": "CNAME"})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

func TestCreateRecord_InvalidJSON(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/zones/z1/records", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestCreateRecord_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		createDNSRecordFunc: func(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			return cloudflare.DNSRecord{}, errors.New("cf error")
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones/z1/records", map[string]any{
		"type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1,
	})
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /zones/{zoneID}/records/{recordID} ---

func TestGetRecord_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error) {
			return cloudflare.DNSRecord{ID: "r1", Type: "A", Name: "test.example.com", Content: "1.2.3.4"}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones/z1/records/r1", true)
	assertStatus(t, resp, http.StatusOK)
}

func TestGetRecord_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error) {
			return cloudflare.DNSRecord{}, errors.New("not found")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones/z1/records/r1", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- PUT /zones/{zoneID}/records/{recordID} ---

func TestUpdateRecord_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		updateDNSRecordFunc: func(ctx context.Context, zoneID, recordID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			rec.ID = recordID
			return rec, nil
		},
	})
	defer srv.Close()
	resp := putJSON(t, srv, "/zones/z1/records/r1", map[string]any{
		"type": "A", "name": "test.example.com", "content": "5.6.7.8", "ttl": 300,
	})
	assertStatus(t, resp, http.StatusOK)
}

func TestUpdateRecord_InvalidJSON(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/zones/z1/records/r1", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestUpdateRecord_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		updateDNSRecordFunc: func(ctx context.Context, zoneID, recordID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			return cloudflare.DNSRecord{}, errors.New("cf error")
		},
	})
	defer srv.Close()
	resp := putJSON(t, srv, "/zones/z1/records/r1", map[string]any{
		"type": "A", "name": "test.example.com", "content": "5.6.7.8", "ttl": 300,
	})
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestUpdateRecord_MissingFields(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	resp := putJSON(t, srv, "/zones/z1/records/r1", map[string]any{"type": "A"})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

// --- DELETE /zones/{zoneID}/records/{recordID} ---

func TestDeleteRecord_OK(t *testing.T) {
	var gotZone, gotRecord string
	srv := newTestServer(t, &mockCloudflare{
		deleteDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) error {
			gotZone, gotRecord = zoneID, recordID
			return nil
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones/z1/records/r1")
	assertStatus(t, resp, http.StatusNoContent)
	if gotZone != "z1" || gotRecord != "r1" {
		t.Errorf("delete called with (%q, %q)", gotZone, gotRecord)
	}
}

func TestDeleteRecord_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		deleteDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) error {
			return errors.New("cf error")
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones/z1/records/r1")
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /zones-by-name/{zoneName}/records ---

func TestListRecordsByZoneName_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			if name != "example.com" {
				t.Errorf("zone name = %q, want example.com", name)
			}
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return []cloudflare.DNSRecord{{ID: "r1"}}, nil
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones-by-name/example.com/records", true)
	assertStatus(t, resp, http.StatusOK)
}

func TestListRecordsByZoneName_UpstreamListError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return nil, errors.New("cloudflare unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones-by-name/example.com/records", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestListRecordsByZoneName_ZoneNotFound(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "", errors.New("not found")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/zones-by-name/unknown.com/records", true)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

// --- POST /zones-by-name/{zoneName}/records ---

func TestCreateRecordByZoneName_OK(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		createDNSRecordFunc: func(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			rec.ID = "new-id"
			return rec, nil
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones-by-name/example.com/records", map[string]any{
		"type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1,
	})
	assertStatus(t, resp, http.StatusCreated)
}

func TestCreateRecordByZoneName_ZoneNotFound(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "", errors.New("not found")
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones-by-name/unknown.com/records", map[string]any{
		"type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1,
	})
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestCreateRecordByZoneName_InvalidJSON(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
	})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/zones-by-name/example.com/records", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestCreateRecordByZoneName_MissingFields(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones-by-name/example.com/records", map[string]any{"type": "CNAME"})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

func TestCreateRecordByZoneName_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		createDNSRecordFunc: func(ctx context.Context, zoneID string, rec cloudflare.DNSRecord) (cloudflare.DNSRecord, error) {
			return cloudflare.DNSRecord{}, errors.New("cf error")
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/zones-by-name/example.com/records", map[string]any{
		"type": "CNAME", "name": "test.example.com", "content": "example.com", "ttl": 1,
	})
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- DELETE /zones-by-name/{zoneName}/records/{recordName} ---

func TestDeleteRecordByZoneName_OK(t *testing.T) {
	var deletedID string
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return []cloudflare.DNSRecord{{ID: "r1", Name: "test.example.com"}}, nil
		},
		deleteDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) error {
			deletedID = recordID
			return nil
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/example.com/records/test.example.com")
	assertStatus(t, resp, http.StatusNoContent)
	if deletedID != "r1" {
		t.Errorf("deleted record ID = %q, want r1", deletedID)
	}
}

func TestDeleteRecordByZoneName_ZoneNotFound(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "", errors.New("not found")
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/unknown.com/records/test.example.com")
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestDeleteRecordByZoneName_ListError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return nil, errors.New("cf error")
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/example.com/records/test.example.com")
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestDeleteRecordByZoneName_DeleteUpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return []cloudflare.DNSRecord{{ID: "r1", Name: "test.example.com"}}, nil
		},
		deleteDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) error {
			return errors.New("cf delete error")
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/example.com/records/test.example.com")
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestDeleteRecordByZoneName_MultipleRecords(t *testing.T) {
	var deletedIDs []string
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return []cloudflare.DNSRecord{
				{ID: "r1", Name: "test.example.com"},
				{ID: "r2", Name: "test.example.com"},
			}, nil
		},
		deleteDNSRecordFunc: func(ctx context.Context, zoneID, recordID string) error {
			deletedIDs = append(deletedIDs, recordID)
			return nil
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/example.com/records/test.example.com")
	assertStatus(t, resp, http.StatusNoContent)
	if len(deletedIDs) != 2 {
		t.Errorf("deleted %d records, want 2", len(deletedIDs))
	}
}

func TestDeleteRecordByZoneName_RecordNotFound(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{
		getZoneIDByNameFunc: func(ctx context.Context, name string) (string, error) {
			return "z1", nil
		},
		listDNSRecordsFunc: func(ctx context.Context, zoneID, rt, rn string) ([]cloudflare.DNSRecord, error) {
			return []cloudflare.DNSRecord{}, nil
		},
	})
	defer srv.Close()
	resp := deleteReq(t, srv, "/zones-by-name/example.com/records/missing.example.com")
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

// --- X-Trace-ID ---

func TestTraceID_Generated(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	// The request logger should generate a trace ID if absent — no 500 error
	resp := getJSON(t, srv, "/health", false)
	if resp.StatusCode == http.StatusInternalServerError {
		t.Error("expected no error when X-Trace-ID is absent")
	}
}

func TestTraceID_Propagated(t *testing.T) {
	srv := newTestServer(t, &mockCloudflare{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	req.Header.Set("X-Trace-ID", "my-trace-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode == http.StatusInternalServerError {
		t.Error("expected no error with X-Trace-ID header")
	}
}
