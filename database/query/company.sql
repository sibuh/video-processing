-- name: CreateCompany :one
INSERT INTO companies (
    name,
    latitude,
    longitude,
    location_name
) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetCompany :one
SELECT * FROM companies WHERE id = $1;

-- name: ListCompanies :many
SELECT * FROM companies ORDER BY created_at DESC;

-- name: UpdateCompany :one
UPDATE companies
SET name = COALESCE(NULLIF($1, name), name),
    latitude = COALESCE($2, latitude),
    longitude = COALESCE($3, longitude),
    location_name = COALESCE(NULLIF($4, location_name), location_name)
WHERE id = $5 RETURNING *;

-- name: DeleteCompany :one
DELETE FROM companies WHERE id = $1 RETURNING *;
