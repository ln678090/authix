package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"git.blinkpay.vn/software/authix/internal/persistence"
	// "github.com/rs/zerolog/log"
)

func (p *Persister) InsertPermissionWithMask(
	ctx context.Context,
	sourceID string,
	sourceType string,
	deviceID string,
	permissionMask int16,
	inheritsFrom []string,
) (*persistence.PermissionAssignment, error) {
	if permissionMask < 0 || permissionMask > 15 {
		return nil, fmt.Errorf("permission_mask must be between 0 and 15")
	}

	if inheritsFrom == nil {
		inheritsFrom = []string{}
	}

	inheritsFromStr := "{" + strings.Join(inheritsFrom, ",") + "}"

	perm := &persistence.PermissionAssignment{}

	query := `
    INSERT INTO permission_assignments (
        source_id,
        source_type,
        device_id,
        permission_mask,
        inherits_from
    )
    VALUES (?, ?, ?, ?, ?::uuid[])
    RETURNING
        id,
        source_id,
        source_type,
        device_id,
        permission_mask,
        inherits_from::text
`

	var inheritsFromDB string

	err := p.conn.NewRaw(
		query,
		sourceID,
		sourceType,
		deviceID,
		permissionMask,
		inheritsFromStr,
	).Scan(
		ctx,
		&perm.ID,
		&perm.SourceID,
		&perm.SourceType,
		&perm.DeviceID,
		&perm.PermissionMask,
		&inheritsFromDB,
		// &perm.CreatedAt,
		// &perm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	inheritsFromDB = strings.Trim(inheritsFromDB, "{}")
	if inheritsFromDB == "" {
		perm.InheritsFrom = []string{}
	} else {
		perm.InheritsFrom = strings.Split(inheritsFromDB, ",")
	}

	return perm, nil
}

func (p *Persister) GetEffectivePermissions(
	ctx context.Context,
	sourceID string,
	sourceType string,
) (*persistence.EffectivePermissions, error) {
	// get cache
	cacheKey := fmt.Sprintf("%s:%s", sourceType, sourceID)
	if cached, ok := p.cache.Get(cacheKey); ok {
		//	log.Debug().Str("key", cacheKey).Msg("Cache hit")
		return cached, nil
	}

	//log.Debug().Str("key", cacheKey).Msg("Cache miss")

	var resultJSON string

	err := p.conn.NewRaw(
		"SELECT get_effective_permissions(?, ?) AS result",
		sourceID, sourceType,
	).Scan(ctx, &resultJSON)
	if err != nil {
		return nil, err
	}

	var perms persistence.EffectivePermissions
	if err := json.Unmarshal([]byte(resultJSON), &perms); err != nil {
		return nil, err
	}
	// set cache
	p.cache.Set(cacheKey, &perms)

	return &perms, nil
}

func (p *Persister) GetPermission(ctx context.Context, id string) (*persistence.PermissionAssignment, error) {
	perm := &persistence.PermissionAssignment{}

	query := `
    SELECT id, source_id, source_type, device_id, permission_mask, inherits_from, created_at
    FROM permission_assignments
    WHERE id = ?
    `

	row := p.conn.QueryRowContext(ctx, query, id)

	var inheritsFromStr string
	err := row.Scan(
		&perm.ID,
		&perm.SourceID,
		&perm.SourceType,
		&perm.DeviceID,
		&perm.PermissionMask,
		&inheritsFromStr,
	)
	if err != nil {
		return nil, err
	}

	// Parse inherits_from array
	inheritsFromStr = strings.Trim(inheritsFromStr, "{}")
	if inheritsFromStr != "" {
		perm.InheritsFrom = strings.Split(inheritsFromStr, ",")
	} else {
		perm.InheritsFrom = []string{}
	}

	return perm, nil
}

func (p *Persister) UpdatePermission(
	ctx context.Context,
	id string,
	permissionMask int16,
	inheritsFrom []string,
) error {
	if permissionMask < 0 || permissionMask > 15 {
		return sql.ErrNoRows
	}

	if inheritsFrom == nil {
		inheritsFrom = []string{}
	}

	inheritsFromStr := "{" + strings.Join(inheritsFrom, ",") + "}"

	query := `
    UPDATE permission_assignments
    SET permission_mask = ?, inherits_from = ?::uuid[]
    WHERE id = ?
    `

	_, err := p.conn.ExecContext(ctx, query, permissionMask, inheritsFromStr, id)
	return err
}

func (p *Persister) DeletePermission(ctx context.Context, id string) error {
	query := `
    DELETE FROM permission_assignments
    WHERE id = ?
    `

	_, err := p.conn.ExecContext(ctx, query, id)
	return err
}
