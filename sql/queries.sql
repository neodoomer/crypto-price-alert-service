-- name: CreateAlert :one
INSERT INTO alerts (token, target_price, direction, callback_url, callback_secret)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAlerts :many
SELECT id, token, target_price, direction, callback_url, triggered, created_at, updated_at
FROM alerts
WHERE (sqlc.narg('triggered')::BOOLEAN IS NULL OR triggered = sqlc.narg('triggered')::BOOLEAN)
ORDER BY created_at DESC;

-- name: DeleteAlert :execresult
DELETE FROM alerts WHERE id = $1 AND triggered = FALSE;

-- name: GetActiveAlerts :many
SELECT * FROM alerts WHERE triggered = FALSE;

-- name: MarkAlertTriggered :exec
UPDATE alerts SET triggered = TRUE, updated_at = now() WHERE id = $1;
