-- +goose Up
CREATE TABLE users (
    id            bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL UNIQUE CHECK (email = lower(email)),
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash bytea       PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE tasks (
    id         bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      text        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    completed  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tasks_user_id_created_at_idx
    ON tasks (user_id, created_at DESC, id DESC);

CREATE TABLE login_attempts (
    email        text        PRIMARY KEY,
    attempts     integer     NOT NULL,
    window_start timestamptz NOT NULL
);

-- +goose Down
DROP TABLE login_attempts;
DROP TABLE tasks;
DROP TABLE sessions;
DROP TABLE users;
