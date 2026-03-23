package cloudflare

import (
	"context"
	"fmt"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
)

// DNSRecord represents a DNS record returned by or sent to the Cloudflare API.
type DNSRecord struct {
	ID         string    `json:"id,omitempty"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	TTL        int       `json:"ttl"`
	Proxied    *bool     `json:"proxied,omitempty"`
	CreatedOn  time.Time `json:"created_on,omitempty"`
	ModifiedOn time.Time `json:"modified_on,omitempty"`
}

// Zone represents a Cloudflare DNS zone.
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Client wraps the Cloudflare API using the official cloudflare-go library.
type Client struct {
	api *cf.API
}

// NewClient creates a new Cloudflare API client using an API token.
func NewClient(apiToken string) (*Client, error) {
	api, err := cf.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("creating cloudflare client: %w", err)
	}
	return &Client{api: api}, nil
}

// NewClientFromAPI creates a Client from an existing *cf.API instance.
// This is primarily useful for testing with a custom base URL.
func NewClientFromAPI(api *cf.API) *Client {
	return &Client{api: api}
}

// Ping verifies connectivity to the Cloudflare API by listing zones.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.api.ListZones(ctx)
	if err != nil {
		return fmt.Errorf("cloudflare ping failed: %w", err)
	}
	return nil
}

// ListZones returns all zones accessible with the current API token.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	zones, err := c.api.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}
	result := make([]Zone, len(zones))
	for i, z := range zones {
		result[i] = Zone{
			ID:     z.ID,
			Name:   z.Name,
			Status: z.Status,
		}
	}
	return result, nil
}

// GetZoneIDByName returns the zone ID for the given zone name.
func (c *Client) GetZoneIDByName(ctx context.Context, zoneName string) (string, error) {
	id, err := c.api.ZoneIDByName(zoneName)
	if err != nil {
		return "", fmt.Errorf("looking up zone %q: %w", zoneName, err)
	}
	return id, nil
}

// ListDNSRecords returns DNS records for the given zone, optionally filtered by type and name.
func (c *Client) ListDNSRecords(ctx context.Context, zoneID, recordType, recordName string) ([]DNSRecord, error) {
	params := cf.ListDNSRecordsParams{}
	if recordType != "" {
		params.Type = recordType
	}
	if recordName != "" {
		params.Name = recordName
	}
	rc := cf.ZoneIdentifier(zoneID)
	records, _, err := c.api.ListDNSRecords(ctx, rc, params)
	if err != nil {
		return nil, fmt.Errorf("listing DNS records: %w", err)
	}
	result := make([]DNSRecord, len(records))
	for i, r := range records {
		result[i] = toDNSRecord(r)
	}
	return result, nil
}

// GetDNSRecord returns a single DNS record by ID.
func (c *Client) GetDNSRecord(ctx context.Context, zoneID, recordID string) (DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	record, err := c.api.GetDNSRecord(ctx, rc, recordID)
	if err != nil {
		return DNSRecord{}, fmt.Errorf("getting DNS record %q: %w", recordID, err)
	}
	return toDNSRecord(record), nil
}

// CreateDNSRecord creates a DNS record and returns the created record.
func (c *Client) CreateDNSRecord(ctx context.Context, zoneID string, rec DNSRecord) (DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	params := cf.CreateDNSRecordParams{
		Type:    rec.Type,
		Name:    rec.Name,
		Content: rec.Content,
		TTL:     rec.TTL,
		Proxied: rec.Proxied,
	}
	created, err := c.api.CreateDNSRecord(ctx, rc, params)
	if err != nil {
		return DNSRecord{}, fmt.Errorf("creating DNS record: %w", err)
	}
	return toDNSRecord(created), nil
}

// UpdateDNSRecord updates an existing DNS record and returns the updated record.
func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, rec DNSRecord) (DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	params := cf.UpdateDNSRecordParams{
		ID:      recordID,
		Type:    rec.Type,
		Name:    rec.Name,
		Content: rec.Content,
		TTL:     rec.TTL,
		Proxied: rec.Proxied,
	}
	updated, err := c.api.UpdateDNSRecord(ctx, rc, params)
	if err != nil {
		return DNSRecord{}, fmt.Errorf("updating DNS record %q: %w", recordID, err)
	}
	return toDNSRecord(updated), nil
}

// DeleteDNSRecord deletes a DNS record by ID.
func (c *Client) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	rc := cf.ZoneIdentifier(zoneID)
	err := c.api.DeleteDNSRecord(ctx, rc, recordID)
	if err != nil {
		return fmt.Errorf("deleting DNS record %q: %w", recordID, err)
	}
	return nil
}

func toDNSRecord(r cf.DNSRecord) DNSRecord {
	return DNSRecord{
		ID:         r.ID,
		Type:       r.Type,
		Name:       r.Name,
		Content:    r.Content,
		TTL:        r.TTL,
		Proxied:    r.Proxied,
		CreatedOn:  r.CreatedOn,
		ModifiedOn: r.ModifiedOn,
	}
}
