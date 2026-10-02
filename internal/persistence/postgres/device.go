package postgres

import (
	"context"

	"git.blinkpay.vn/software/authix/internal/persistence"
)

func (p *Persister) GetDevices(ctx context.Context) ([]persistence.Device, error) {
	var devices []persistence.Device

	err := p.conn.NewSelect().
		Table("devices").
		Scan(ctx, &devices)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func (p *Persister) CreateDevice(ctx context.Context, deviceType string, serialNumber string) (*persistence.Device, error) {
	device := &persistence.Device{
		DeviceType:   deviceType,
		SerialNumber: serialNumber,
	}
	_, err := p.conn.NewInsert().Model(device).ModelTableExpr("devices").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (p *Persister) GetDevice(ctx context.Context, id string) (*persistence.Device, error) {
	device := &persistence.Device{}

	err := p.conn.NewSelect().
		Table("devices").
		Where("id = ?", id).
		Scan(ctx, device)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (p *Persister) UpdateDevice(ctx context.Context, id, deviceType, serialNumber string) (*persistence.Device, error) {
	device := &persistence.Device{
		DeviceType:   deviceType,
		SerialNumber: serialNumber,
	}

	_, err := p.conn.NewUpdate().
		Model(device).
		ModelTableExpr("devices").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (p *Persister) DeleteDevice(ctx context.Context, id string) error {
	// Check if device has permissions
	var count int
	err := p.conn.NewSelect().
		Table("permission_assignments").
		Where("device_id = ?", id).
		ColumnExpr("COUNT(*)").
		Scan(ctx, &count)
	if err != nil {
		return err
	}

	if count > 0 {
		return persistence.ErrDeviceHasPermissions
	}

	_, err = p.conn.NewDelete().
		Table("devices").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
