package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"git.blinkpay.vn/software/authix/internal/persistence"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type CreatePermissionRequest struct {
	SourceID       string   `json:"source_id"`
	SourceType     string   `json:"source_type"`
	DeviceID       string   `json:"device_id"`
	PermissionMask int16    `json:"permission_mask"`
	InheritsFrom   []string `json:"inherits_from"`
}

func (h *RESTHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req CreatePermissionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(req.SourceID) == "" ||
		strings.TrimSpace(req.SourceType) == "" ||
		strings.TrimSpace(req.DeviceID) == "" {
		h.respondWithError(
			w,
			http.StatusBadRequest,
			"source_id, source_type, and device_id are required",
		)
		return
	}

	if req.PermissionMask < 0 || req.PermissionMask > 15 {
		h.respondWithError(
			w,
			http.StatusBadRequest,
			"permission_mask must be between 0 and 15",
		)
		return
	}

	permission, err := h.persister.InsertPermissionWithMask(
		r.Context(),
		req.SourceID,
		req.SourceType,
		req.DeviceID,
		req.PermissionMask,
		req.InheritsFrom,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "23505") {
			h.respondWithError(
				w,
				http.StatusConflict,
				"This permission assignment already exists",
			)
			return
		}

		log.Error().
			Err(err).
			Msg("Failed to insert permission")

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"Internal database error",
		)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, permission)
}

type GetPermissionsRequest struct {
	SourceID     string `json:"source_id"`
	SourceType   string `json:"source_type"`
	DeviceType   string `json:"device_type,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
}

func (h *RESTHandler) GetEffectivePermissions(w http.ResponseWriter, r *http.Request) {
	var req GetPermissionsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(req.SourceID) == "" || strings.TrimSpace(req.SourceType) == "" {
		h.respondWithError(w, http.StatusBadRequest, "source_id and source_type are required")
		return
	}

	perms, err := h.persister.GetEffectivePermissions(r.Context(), req.SourceID, req.SourceType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get effective permissions")
		h.respondWithError(w, http.StatusInternalServerError, "Internal database error")
		return
	}

	if req.DeviceType == "" && req.SerialNumber == "" {
		h.respondWithJSON(w, http.StatusOK, perms)
		return
	}

	filtered := filterPermissions(perms, req.DeviceType, req.SerialNumber)
	h.respondWithJSON(w, http.StatusOK, filtered)
}

func filterPermissions(perms *persistence.EffectivePermissions, deviceType, serialNumber string) *persistence.EffectivePermissions {
	filteredA := make([]string, 0, len(perms.A))
	filteredB := make([]string, 0, len(perms.B))

	for i, serial := range perms.A {
		// Filter by serial_number first (more specific)
		if serialNumber != "" {
			if serial == serialNumber {
				filteredA = append(filteredA, serial)
				filteredB = append(filteredB, perms.B[i])
			}
			continue
		}

		// Filter by device_type
		if deviceType != "" {
			deviceTypeFromSerial := getDeviceTypeFromSerial(serial)
			if deviceTypeFromSerial == deviceType {
				filteredA = append(filteredA, serial)
				filteredB = append(filteredB, perms.B[i])
			}
		}
	}

	return &persistence.EffectivePermissions{
		A: filteredA,
		B: filteredB,
	}
}

// Helper: extract device_type from serial_number
// card_scanner1 -> card_scanner
// pressure_washer1 -> pressure_washer
func getDeviceTypeFromSerial(serial string) string {
	// Remove trailing number
	for i := len(serial) - 1; i >= 0; i-- {
		if serial[i] >= '0' && serial[i] <= '9' {
			continue
		}
		return serial[:i+1]
	}
	return serial
}

type UpdatePermissionReq struct {
	PermissionMask int16    `json:"permission_mask"`
	InheritsFrom   []string `json:"inherits_from"`
}

func (h *RESTHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if strings.TrimSpace(id) == "" {
		h.respondWithError(w, http.StatusBadRequest, "permission id is required")
		return
	}

	perm, err := h.persister.GetPermission(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Permission not found")
		return
	}

	h.respondWithJSON(w, http.StatusOK, perm)
}

func (h *RESTHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		h.respondWithError(w, http.StatusBadRequest, "permission id is required")
		return
	}
	var req UpdatePermissionReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if req.PermissionMask < 0 || req.PermissionMask > 15 {
		h.respondWithError(w, http.StatusBadRequest, "permission_mask must be between 0 and 15")
		return
	}
	err := h.persister.UpdatePermission(r.Context(), id, req.PermissionMask, req.InheritsFrom)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update permission")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update permission")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RESTHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if strings.TrimSpace(id) == "" {
		h.respondWithError(w, http.StatusBadRequest, "permission id is required")
		return
	}

	if err := h.persister.DeletePermission(r.Context(), id); err != nil {
		log.Error().
			Err(err).
			Str("permission_id", id).
			Msg("Failed to delete permission")

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"Failed to delete permission",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
