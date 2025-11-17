
-- name: CreateUser :one
INSERT INTO users (
    first_name,
    middle_name,
    last_name,
    phone,
    username,
    password,
    email
) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: SearchUsers :many
SELECT * FROM users WHERE first_name ILIKE $1 
OR last_name ILIKE $1 
OR username ILIKE $1 
OR email ILIKE $1 
OR phone ILIKE $1;

-- name: UpdateUser :one
UPDATE users
SET first_name = COALESCE(NULLIF($1, first_name), first_name),
    middle_name=COALESCE(NULLIF($2, middle_name), middle_name),
    last_name = COALESCE(NULLIF($3, last_name), last_name),
    phone = COALESCE(NULLIF($4, phone), phone),
    username = COALESCE(NULLIF($5, username), username),
    email = COALESCE(NULLIF($6, email), email)
WHERE id = $7 RETURNING *;

-- name: DeleteUser :one
DELETE FROM users WHERE id = $1 RETURNING *;

