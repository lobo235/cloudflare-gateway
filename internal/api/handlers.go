package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lobo235/cloudflare-gateway/internal/cloudflare"
)

// listZonesHandler handles GET /zones.
func (s *Server) listZonesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zones, err := s.cf.ListZones(r.Context())
		if err != nil {
			s.log.Error("failed to list zones", "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to list zones from Cloudflare")
			return
		}
		writeJSON(w, http.StatusOK, zones)
	}
}

// listRecordsHandler handles GET /zones/{zoneID}/records.
// Supports optional query params: ?type=CNAME&name=foo
func (s *Server) listRecordsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneID := r.PathValue("zoneID")
		recordType := r.URL.Query().Get("type")
		recordName := r.URL.Query().Get("name")

		records, err := s.cf.ListDNSRecords(r.Context(), zoneID, recordType, recordName)
		if err != nil {
			s.log.Error("failed to list DNS records", "zone_id", zoneID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to list DNS records from Cloudflare")
			return
		}
		writeJSON(w, http.StatusOK, records)
	}
}

// createRecordHandler handles POST /zones/{zoneID}/records.
func (s *Server) createRecordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneID := r.PathValue("zoneID")

		var body cloudflare.DNSRecord
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
			return
		}
		if body.Type == "" || body.Name == "" || body.Content == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "type, name, and content are required")
			return
		}

		created, err := s.cf.CreateDNSRecord(r.Context(), zoneID, body)
		if err != nil {
			s.log.Error("failed to create DNS record", "zone_id", zoneID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to create DNS record")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

// getRecordHandler handles GET /zones/{zoneID}/records/{recordID}.
func (s *Server) getRecordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneID := r.PathValue("zoneID")
		recordID := r.PathValue("recordID")

		record, err := s.cf.GetDNSRecord(r.Context(), zoneID, recordID)
		if err != nil {
			s.log.Error("failed to get DNS record", "zone_id", zoneID, "record_id", recordID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("failed to get DNS record %q", recordID))
			return
		}
		writeJSON(w, http.StatusOK, record)
	}
}

// updateRecordHandler handles PUT /zones/{zoneID}/records/{recordID}.
func (s *Server) updateRecordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneID := r.PathValue("zoneID")
		recordID := r.PathValue("recordID")

		var body cloudflare.DNSRecord
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
			return
		}
		if body.Type == "" || body.Name == "" || body.Content == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "type, name, and content are required")
			return
		}

		updated, err := s.cf.UpdateDNSRecord(r.Context(), zoneID, recordID, body)
		if err != nil {
			s.log.Error("failed to update DNS record", "zone_id", zoneID, "record_id", recordID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("failed to update DNS record %q", recordID))
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

// deleteRecordHandler handles DELETE /zones/{zoneID}/records/{recordID}.
func (s *Server) deleteRecordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneID := r.PathValue("zoneID")
		recordID := r.PathValue("recordID")

		if err := s.cf.DeleteDNSRecord(r.Context(), zoneID, recordID); err != nil {
			s.log.Error("failed to delete DNS record", "zone_id", zoneID, "record_id", recordID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("failed to delete DNS record %q", recordID))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// listRecordsByZoneNameHandler handles GET /zones/name/{zoneName}/records.
func (s *Server) listRecordsByZoneNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneName := r.PathValue("zoneName")

		zoneID, err := s.cf.GetZoneIDByName(r.Context(), zoneName)
		if err != nil {
			s.log.Error("failed to resolve zone name", "zone_name", zoneName, "error", err, "trace_id", tid)
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("zone %q not found", zoneName))
			return
		}

		recordType := r.URL.Query().Get("type")
		recordName := r.URL.Query().Get("name")
		records, err := s.cf.ListDNSRecords(r.Context(), zoneID, recordType, recordName)
		if err != nil {
			s.log.Error("failed to list DNS records", "zone_id", zoneID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to list DNS records from Cloudflare")
			return
		}
		writeJSON(w, http.StatusOK, records)
	}
}

// createRecordByZoneNameHandler handles POST /zones/name/{zoneName}/records.
func (s *Server) createRecordByZoneNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneName := r.PathValue("zoneName")

		zoneID, err := s.cf.GetZoneIDByName(r.Context(), zoneName)
		if err != nil {
			s.log.Error("failed to resolve zone name", "zone_name", zoneName, "error", err, "trace_id", tid)
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("zone %q not found", zoneName))
			return
		}

		var body cloudflare.DNSRecord
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
			return
		}
		if body.Type == "" || body.Name == "" || body.Content == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "type, name, and content are required")
			return
		}

		created, err := s.cf.CreateDNSRecord(r.Context(), zoneID, body)
		if err != nil {
			s.log.Error("failed to create DNS record", "zone_id", zoneID, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to create DNS record")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

// deleteRecordByZoneNameHandler handles DELETE /zones/name/{zoneName}/records/{recordName}.
// Looks up the zone by name, finds the record by name, and deletes it.
func (s *Server) deleteRecordByZoneNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid := traceID(r.Context())
		zoneName := r.PathValue("zoneName")
		recordName := r.PathValue("recordName")

		zoneID, err := s.cf.GetZoneIDByName(r.Context(), zoneName)
		if err != nil {
			s.log.Error("failed to resolve zone name", "zone_name", zoneName, "error", err, "trace_id", tid)
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("zone %q not found", zoneName))
			return
		}

		// Find the record by name in the zone
		records, err := s.cf.ListDNSRecords(r.Context(), zoneID, "", recordName)
		if err != nil {
			s.log.Error("failed to list DNS records for delete", "zone_id", zoneID, "record_name", recordName, "error", err, "trace_id", tid)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to look up DNS record")
			return
		}
		if len(records) == 0 {
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no DNS record named %q found in zone %q", recordName, zoneName))
			return
		}

		// Delete all matching records (there could be multiple types for the same name)
		for _, rec := range records {
			if err := s.cf.DeleteDNSRecord(r.Context(), zoneID, rec.ID); err != nil {
				s.log.Error("failed to delete DNS record", "zone_id", zoneID, "record_id", rec.ID, "error", err, "trace_id", tid)
				writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("failed to delete DNS record %q", rec.ID))
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
