-- name: CreateGoogleDriveWatch :one
INSERT INTO google_drive_watch (
    channel_id, resource_id, expires_at, webhook_url
) VALUES ( $1, $2, $3, $4)
RETURNING *;

-- name: UpdateGoogleDriveWatch :one
UPDATE google_drive_watch
SET channel_id = $2,
    resource_id = $3,
    expires_at = $4,
    webhook_url = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetLatestGoogleDriveWatch :one
SELECT * FROM google_drive_watch
ORDER BY created_at DESC
LIMIT 1;

-- name: GetWatchEntriesByFolderIDs :many
SELECT DISTINCT ON (resource_id) *
FROM google_drive_watch
WHERE resource_id = ANY(sqlc.arg(resource_ids)::text[])
ORDER BY resource_id, created_at DESC;
