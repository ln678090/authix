package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"git.blinkpay.vn/software/authix/internal/persistence"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type CreateDeviceReq struct {
	DeviceType   string `json:"device_type"`
	SerialNumber string `json:"serial_number"`
}

var validDeviceTypes = map[string]bool{
	"card_scanner":    true,
	"pressure_washer": true,
	"thermal_printer": true,
	"barcode_scanner": true,
}

func isValidDeviceType(deviceType string) bool {
	return validDeviceTypes[deviceType]
}

func (h *RESTHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(req.DeviceType) == "" || strings.TrimSpace(req.SerialNumber) == "" {
		h.respondWithError(w, http.StatusBadRequest, "device_type and serial_number are required")
		return
	}

	if !isValidDeviceType(req.DeviceType) {
		h.respondWithError(w, http.StatusBadRequest, "Invalid device_type")
		return
	}

	device, err := h.persister.CreateDevice(r.Context(), req.DeviceType, req.SerialNumber)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create device")
		h.respondWithError(w, http.StatusInternalServerError, "Internal database error")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, device)
}

func (h *RESTHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.persister.GetDevices(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to get devices")
		h.respondWithError(w, http.StatusInternalServerError, "Internal database error")
		return
	}
	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
	})
}

func (h *RESTHandler) GetDeviceByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	device, err := h.persister.GetDevice(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Device not found")
		return
	}

	h.respondWithJSON(w, http.StatusOK, device)
}

func (h *RESTHandler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		DeviceType   string `json:"device_type"`
		SerialNumber string `json:"serial_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(req.DeviceType) == "" || strings.TrimSpace(req.SerialNumber) == "" {
		h.respondWithError(w, http.StatusBadRequest, "device_type and serial_number are required")
		return
	}

	if !isValidDeviceType(req.DeviceType) {
		h.respondWithError(w, http.StatusBadRequest, "Invalid device_type")
		return
	}

	device, err := h.persister.UpdateDevice(r.Context(), id, req.DeviceType, req.SerialNumber)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update device")
		return
	}

	h.respondWithJSON(w, http.StatusOK, device)
}

func (h *RESTHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.persister.DeleteDevice(r.Context(), id)
	if err != nil {
		if err == persistence.ErrDeviceHasPermissions {
			h.respondWithError(w, http.StatusBadRequest, "Cannot delete device with active permissions")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
