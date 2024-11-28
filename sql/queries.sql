-- name: GetSchema :one
SELECT * FROM entity_schemas
WHERE name = ? AND deleted_at IS NULL;

-- name: CreateSchema :exec
INSERT INTO entity_schemas (
    id, name, description, fields, indexes
) VALUES (?, ?, ?, ?, ?);

-- name: ListSchemas :many
SELECT * FROM entity_schemas
WHERE deleted_at IS NULL;

-- name: UpdateSchema :exec
UPDATE entity_schemas
SET description = ?,
    fields = ?,
    indexes = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE name = ? AND deleted_at IS NULL;

-- name: SoftDeleteSchema :exec
UPDATE entity_schemas
SET deleted_at = CURRENT_TIMESTAMP
WHERE name = ? AND deleted_at IS NULL;
