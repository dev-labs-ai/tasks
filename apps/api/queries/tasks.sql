-- Every task query is scoped by user_id: another user's task matches no
-- rows, which the service reports as not found (tech spec §5.3).

-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = sqlc.arg('user_id')
  AND (sqlc.narg('completed')::boolean IS NULL
       OR completed = sqlc.narg('completed'))
ORDER BY created_at DESC, id DESC;

-- name: CreateTask :one
INSERT INTO tasks (user_id, title)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET title      = coalesce(sqlc.narg('title'), title),
    completed  = coalesce(sqlc.narg('completed'), completed),
    updated_at = now()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id');
