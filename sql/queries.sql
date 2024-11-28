-- name: GetSchema :one
SELECT * FROM entity_schemas
WHERE name = ? AND deleted_at IS NULL;

-- name: CreateSchema :exec
INSERT INTO entity_schemas (
    id, name, description, fields, indexes, annotations
) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListSchemas :many
SELECT * FROM entity_schemas
WHERE deleted_at IS NULL;

-- name: UpdateSchema :exec
UPDATE entity_schemas
SET description = ?,
    fields = ?,
    indexes = ?,
    annotations = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE name = ? AND deleted_at IS NULL;

-- name: SoftDeleteSchema :exec
UPDATE entity_schemas
SET deleted_at = CURRENT_TIMESTAMP
WHERE name = ? AND deleted_at IS NULL;

-- name: GetLatestVersion :one
SELECT version FROM schema_versions
WHERE schema_name = ? 
ORDER BY version DESC
LIMIT 1;

-- name: AddSchemaVersion :exec
INSERT INTO schema_versions (
    schema_name, version, changes
) VALUES (?, ?, ?);

-- name: AddSchemaLog :exec
INSERT INTO schema_logs (
    schema_name, operation, operator, details
) VALUES (?, ?, ?, ?);

-- name: GetSchemaLogs :many
SELECT * FROM schema_logs
WHERE schema_name = ?
ORDER BY timestamp DESC;

-- name: AddSchemaDependency :exec
INSERT INTO schema_dependencies (
    parent_schema, child_schema, dependency_type
) VALUES (?, ?, ?);

-- name: GetSchemaDependencies :many
SELECT * FROM schema_dependencies
WHERE parent_schema = ? OR child_schema = ?;
