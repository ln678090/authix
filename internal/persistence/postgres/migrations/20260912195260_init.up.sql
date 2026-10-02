BEGIN;



CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    device_type VARCHAR(100) NOT NULL,
    serial_number VARCHAR(100) NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_serial_number
ON devices (serial_number);

CREATE TABLE IF NOT EXISTS permission_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    source_id UUID NOT NULL,
    source_type VARCHAR(100) NOT NULL,

    device_id UUID NOT NULL
        REFERENCES devices(id)
        ON DELETE CASCADE,

    permission_mask SMALLINT NOT NULL,

    inherits_from UUID[] NOT NULL DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uniq_source_device
        UNIQUE (source_id, source_type, device_id),

    CONSTRAINT permission_mask_range
        CHECK (permission_mask >= 0 AND permission_mask <= 15)
);

CREATE INDEX IF NOT EXISTS idx_permission_assignments_device
ON permission_assignments (device_id);

CREATE INDEX IF NOT EXISTS idx_permission_assignments_source
ON permission_assignments (source_id, source_type);

CREATE INDEX IF NOT EXISTS idx_permission_assignments_inherits_from_gin
ON permission_assignments
USING GIN (inherits_from);


CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_devices_updated_at ON devices;

CREATE TRIGGER trg_devices_updated_at
BEFORE UPDATE ON devices
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_permission_assignments_updated_at
ON permission_assignments;

CREATE TRIGGER trg_permission_assignments_updated_at
BEFORE UPDATE ON permission_assignments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


CREATE OR REPLACE FUNCTION notify_devices_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_notify(
        'permission_cache_invalidate',
        json_build_object(
            'type', 'devices',
            'operation', TG_OP,
            'device_id', COALESCE(NEW.id, OLD.id)
        )::text
    );

    RETURN COALESCE(NEW, OLD);
END;
$$;

CREATE TRIGGER trg_devices_notify
AFTER INSERT OR UPDATE OF serial_number
ON devices
FOR EACH ROW
EXECUTE FUNCTION notify_devices_change();


CREATE OR REPLACE FUNCTION notify_permission_assignments_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_source_id UUID;
    v_source_type VARCHAR;
    v_device_id UUID;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_source_id := OLD.source_id;
        v_source_type := OLD.source_type;
        v_device_id := OLD.device_id;
    ELSE
        v_source_id := NEW.source_id;
        v_source_type := NEW.source_type;
        v_device_id := NEW.device_id;
    END IF;

    PERFORM pg_notify(
        'permission_cache_invalidate',
        json_build_object(
            'type', 'permission_assignments',
            'operation', TG_OP,
            'source_id', v_source_id,
            'source_type', v_source_type,
            'device_id', v_device_id
        )::text
    );

    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS trg_permission_assignments_notify
ON permission_assignments;

CREATE TRIGGER trg_permission_assignments_notify
AFTER INSERT OR UPDATE OR DELETE
ON permission_assignments
FOR EACH ROW
EXECUTE FUNCTION notify_permission_assignments_change();

CREATE OR REPLACE FUNCTION get_affected_permission_sources(
    p_source_id UUID,
    p_source_type VARCHAR
)
RETURNS TABLE (
    source_id UUID,
    source_type VARCHAR
)
LANGUAGE sql
STABLE
AS $$
    WITH RECURSIVE affected_sources AS (
        SELECT
            p_source_id AS source_id,
            p_source_type AS source_type

        UNION

        SELECT
            pa.source_id,
            pa.source_type
        FROM permission_assignments pa
        JOIN affected_sources affected
          ON affected.source_id = ANY(pa.inherits_from)
    )
    SELECT DISTINCT
        affected_sources.source_id,
        affected_sources.source_type
    FROM affected_sources;
$$;


CREATE OR REPLACE FUNCTION get_effective_permissions(
    p_source_id UUID,
    p_source_type VARCHAR
)
RETURNS JSON
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    result JSON;
BEGIN
    WITH RECURSIVE inherited_sources AS (
        -- Source được request
        SELECT
            pa.source_id,
            pa.source_type,
            pa.inherits_from,
            ARRAY[pa.source_id] AS visited
        FROM permission_assignments pa
        WHERE pa.source_id = p_source_id
          AND pa.source_type = p_source_type

        UNION

        SELECT
            pa.source_id,
            pa.source_type,
            pa.inherits_from,
            src.visited || pa.source_id
        FROM permission_assignments pa
        JOIN inherited_sources src
          ON pa.source_id = ANY(src.inherits_from)
        WHERE NOT pa.source_id = ANY(src.visited)
    ),

    unique_sources AS (
        SELECT DISTINCT source_id, source_type
        FROM inherited_sources
    ),

    effective_permissions AS (
        SELECT
            pa.device_id,
            bit_or(pa.permission_mask) AS permission_mask
        FROM permission_assignments pa
        JOIN unique_sources src
          ON pa.source_id = src.source_id
         AND pa.source_type = src.source_type
        GROUP BY pa.device_id
    )

    SELECT json_build_object(
        'a', COALESCE(
            json_agg(d.serial_number ORDER BY d.serial_number),
            '[]'::json
        ),
        'b', COALESCE(
            json_agg(ep.permission_mask::text ORDER BY d.serial_number),
            '[]'::json
        )
    )
    INTO result
    FROM effective_permissions ep
    JOIN devices d ON d.id = ep.device_id;

    RETURN COALESCE(result, '{"a":[],"b":[]}'::json);
END;
$$;

COMMIT;
