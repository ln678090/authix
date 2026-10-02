package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type invalidationPayload struct {
	Type       string `json:"type"`
	Operation  string `json:"operation"`
	SourceID   string `json:"source_id"`
	SourceType string `json:"source_type"`
	DeviceID   string `json:"device_id"`
}

type permissionSource struct {
	SourceID   string `bun:"source_id"`
	SourceType string `bun:"source_type"`
}

func (p *Persister) listenForInvalidation(ctx context.Context) {
	conn, err := p.pgxPool.Acquire(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to acquire connection for LISTEN")
		return
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN permission_cache_invalidate"); err != nil {
		log.Error().Err(err).Msg("Failed to LISTEN permission_cache_invalidate")
		return
	}

	log.Info().Msg("Listening for permission cache invalidation")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			log.Error().
				Err(err).
				Msg("Error waiting for permission cache notification")

			time.Sleep(time.Second)
			continue
		}

		var payload invalidationPayload
		if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
			log.Error().
				Err(err).
				Str("payload", notification.Payload).
				Msg("Failed to parse permission cache notification")
			continue
		}

		switch payload.Type {
		case "permission_assignments":
			if payload.SourceID == "" || payload.SourceType == "" {
				log.Warn().
					Str("payload", notification.Payload).
					Msg("Permission invalidation ignored: missing source_id/source_type")
				continue
			}

			p.invalidatePermissionCache(
				ctx,
				payload.SourceID,
				payload.SourceType,
			)

		case "devices":
			if payload.DeviceID == "" {
				log.Warn().
					Str("payload", notification.Payload).
					Msg("Device invalidation ignored: missing device_id")
				continue
			}

			p.invalidateDevicePermissionCache(ctx, payload.DeviceID)

		default:
			log.Debug().
				Str("type", payload.Type).
				Str("payload", notification.Payload).
				Msg("Ignoring unknown cache invalidation notification")
		}
	}
}

func (p *Persister) invalidatePermissionCache(
	ctx context.Context,
	sourceID string,
	sourceType string,
) {
	var sources []permissionSource

	err := p.conn.NewRaw(`
        SELECT source_id, source_type
        FROM get_affected_permission_sources(?::uuid, ?::varchar)
    `, sourceID, sourceType).Scan(ctx, &sources)
	if err != nil {
		log.Error().
			Err(err).
			Str("source_id", sourceID).
			Str("source_type", sourceType).
			Msg("Failed to find affected permission cache keys")

		key := fmt.Sprintf("%s:%s", sourceType, sourceID)
		p.cache.Delete(key)
		return
	}

	for _, item := range sources {
		key := fmt.Sprintf("%s:%s", item.SourceType, item.SourceID)
		p.cache.Delete(key)

		// log.Debug().
		// 	Str("key", key).
		// 	Msg("Permission cache key invalidated")
	}

	// log.Info().
	// 	Str("source_id", sourceID).
	// 	Str("source_type", sourceType).
	// 	Int("keys", len(sources)).
	// 	Msg("Permission cache invalidated with inherited descendants")
}

func (p *Persister) invalidateDevicePermissionCache(
	ctx context.Context,
	deviceID string,
) {
	var sources []permissionSource

	err := p.conn.NewRaw(`
        WITH RECURSIVE affected_sources AS (
            SELECT DISTINCT
                pa.source_id,
                pa.source_type
            FROM permission_assignments pa
            WHERE pa.device_id = ?::uuid

            UNION

            SELECT
                pa.source_id,
                pa.source_type
            FROM permission_assignments pa
            JOIN affected_sources affected
              ON affected.source_id = ANY(pa.inherits_from)
        )
        SELECT source_id, source_type
        FROM affected_sources
    `, deviceID).Scan(ctx, &sources)
	if err != nil {
		log.Error().
			Err(err).
			Str("device_id", deviceID).
			Msg("Failed to find affected cache keys for device")
		return
	}

	for _, item := range sources {
		key := fmt.Sprintf("%s:%s", item.SourceType, item.SourceID)
		p.cache.Delete(key)

		log.Debug().
			Str("key", key).
			Str("device_id", deviceID).
			Msg("Device-related permission cache key invalidated")
	}

	// log.Info().
	// 	Str("device_id", deviceID).
	// 	Int("keys", len(sources)).
	// 	Msg("Device-related permission cache invalidated")
}
