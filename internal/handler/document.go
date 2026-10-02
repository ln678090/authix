package handler

import (
	"fmt"
	"net/http"
)

type PermissionExplanation struct {
	Mask        int16    `json:"mask"`
	Binary      string   `json:"binary"`
	Permissions []string `json:"permissions"`
}

type PermissionMaskInfo struct {
	MinValue    int16                   `json:"min_value"`
	MaxValue    int16                   `json:"max_value"`
	BitCount    int                     `json:"bit_count"`
	Permissions []string                `json:"permissions"`
	Examples    []PermissionExplanation `json:"examples"`
}

func (h *RESTHandler) GetPermissionMaskInfo(w http.ResponseWriter, r *http.Request) {
	permissionNames := []string{
		"read",
		"write",
		"execute",
		"admin",
	}
	examples := make([]PermissionExplanation, 16)
	for mask := int16(0); mask < 16; mask++ {
		perms := []string{}
		for i := 0; i < 4; i++ {
			if (mask>>i)&1 == 1 {
				perms = append(perms, permissionNames[i])
			}
		}
		bina := fmt.Sprintf("%04b", mask)

		examples[mask] = PermissionExplanation{
			Mask:        mask,
			Binary:      bina,
			Permissions: perms,
		}
	}
	info := PermissionMaskInfo{
		MinValue:    0,
		MaxValue:    15,
		BitCount:    4,
		Permissions: permissionNames,
		Examples:    examples,
	}
	h.respondWithJSON(w, http.StatusOK, info)
}
