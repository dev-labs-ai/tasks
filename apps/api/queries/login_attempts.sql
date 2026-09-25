-- The login window is 15 minutes. It is defined only in these queries:
-- the service reads window_end instead of repeating the duration.

-- name: RegisterLoginAttempt :one
INSERT INTO login_attempts (email, attempts, window_start)
VALUES (sqlc.arg('email'), 1, now())
ON CONFLICT (email) DO UPDATE
SET attempts     = CASE WHEN login_attempts.window_start <= now() - interval '15 minutes'
                        THEN 1 ELSE login_attempts.attempts + 1 END,
    window_start = CASE WHEN login_attempts.window_start <= now() - interval '15 minutes'
                        THEN now() ELSE login_attempts.window_start END
RETURNING attempts, (window_start + interval '15 minutes')::timestamptz AS window_end;

-- name: ClearLoginAttempts :exec
DELETE FROM login_attempts
WHERE email = $1;

-- name: DeleteStaleLoginAttempts :execrows
DELETE FROM login_attempts
WHERE window_start <= now() - interval '15 minutes';
