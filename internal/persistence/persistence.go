package persistence

import (
	"context"
	"errors"
	"time"
)

var ErrDeviceHasPermissions = errors.New("cannot delete device with active permissions")

type Device struct {
	ID           string    `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	DeviceType   string    `json:"device_type" bun:"device_type,notnull"`
	SerialNumber string    `json:"serial_number" bun:"serial_number,notnull,unique"`
	CreatedAt    time.Time `json:"created_at" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `json:"updated_at" bun:"updated_at,notnull,default:current_timestamp"`
}

type PermissionAssignment struct {
	ID             string   `json:"id"`
	SourceID       string   `json:"source_id"`
	SourceType     string   `json:"source_type"`
	DeviceID       string   `json:"device_id"`
	PermissionMask int16    `json:"permission_mask"`
	InheritsFrom   []string `json:"inherits_from"`
}

type EffectivePermissions struct {
	A []string `json:"a"`
	B []string `json:"b"`
}

type Persister interface {
	InsertPermissionWithMask(
		ctx context.Context,
		sourceID string,
		sourceType string,
		deviceID string,
		permissionMask int16,
		inheritsFrom []string,
	) (*PermissionAssignment, error)
	GetEffectivePermissions(ctx context.Context, sourceID string, sourceType string) (*EffectivePermissions, error)
	GetPermission(ctx context.Context, id string) (*PermissionAssignment, error)
	UpdatePermission(ctx context.Context, id string, PermissionMask int16, inheritsFrom []string) error
	DeletePermission(ctx context.Context, id string) error

	GetDevices(ctx context.Context) ([]Device, error)
	CreateDevice(ctx context.Context, deviceType string, serialNumber string) (*Device, error)
	GetDevice(ctx context.Context, id string) (*Device, error)
	UpdateDevice(ctx context.Context, id, deviceType, serialNumber string) (*Device, error)
	DeleteDevice(ctx context.Context, id string) error
}
