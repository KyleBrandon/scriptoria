-- name: CreateGoogleDriveWatch :one
INSERT INTO google_drive_watch (
    channel_id, resource_id, expires_at, webhook_url
) VALUES ( $1, $2, $3, $4)
RETURNING *;


-- name: GetLatestGoogleDriveWatch :one
SELECT * FROM google_drive_watch
ORDER BY created_at DESC
LIMIT 1;

