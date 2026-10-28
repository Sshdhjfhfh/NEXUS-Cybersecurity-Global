// Nexus IA v6.0 — Portal de clientes multi-tenant (código original de NEXUS).
// Cada organización (tenant) y sus datos están aislados por org_id; todas las
// lecturas del portal filtran SIEMPRE por org_id para que un cliente nunca vea
// datos de otro.
package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrEmailTaken se devuelve cuando el email ya está registrado.
var ErrEmailTaken = errors.New("email already registered")

type Org struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type PortalUser struct {
	ID           int64  `json:"id"`
	OrgID        int64  `json:"org_id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type PortalRequest struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"org_id"`
	Title     string    `json:"title"`
	Scope     string    `json:"scope"`
	Status    string    `json:"status"` // new | scheduled | in_progress | done
	CreatedBy int64     `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

const portalSchema = `
CREATE TABLE IF NOT EXISTS orgs (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS portal_users (
    id            BIGSERIAL PRIMARY KEY,
    org_id        BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'owner',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_users_org ON portal_users(org_id);
CREATE TABLE IF NOT EXISTS portal_requests (
    id         BIGSERIAL PRIMARY KEY,
    org_id     BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    scope      TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'new',
    created_by BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_requests_org ON portal_requests(org_id);
`

// EnsurePortalTables crea las tablas del portal si no existen (aditivo, reversible).
func (d *DB) EnsurePortalTables() error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := coordinateWithSchemaMigration(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(portalSchema); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateOrgWithOwner crea una organización y su primer usuario (owner) en una
// transacción. Devuelve ErrEmailTaken si el email ya existe.
func (d *DB) CreateOrgWithOwner(orgName, email, passwordHash string) (orgID, userID int64, err error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	if err = tx.QueryRow(`INSERT INTO orgs(name) VALUES ($1) RETURNING id`, orgName).Scan(&orgID); err != nil {
		return 0, 0, err
	}
	err = tx.QueryRow(
		`INSERT INTO portal_users(org_id, email, password_hash, role) VALUES ($1,$2,$3,'owner') RETURNING id`,
		orgID, email, passwordHash,
	).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, 0, ErrEmailTaken
		}
		return 0, 0, err
	}
	return orgID, userID, tx.Commit()
}

// GetPortalUserByEmail devuelve el usuario (o nil si no existe).
func (d *DB) GetPortalUserByEmail(email string) (*PortalUser, error) {
	var u PortalUser
	err := d.QueryRow(
		`SELECT id, org_id, email, password_hash, role FROM portal_users WHERE email = $1`, email,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetOrg devuelve la organización por id.
func (d *DB) GetOrg(id int64) (*Org, error) {
	var o Org
	err := d.QueryRow(`SELECT id, name, created_at FROM orgs WHERE id = $1`, id).Scan(&o.ID, &o.Name, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// CreatePortalRequest crea una solicitud de evaluación para la organización dada.
func (d *DB) CreatePortalRequest(orgID, userID int64, title, scope string) (int64, error) {
	var id int64
	err := d.QueryRow(
		`INSERT INTO portal_requests(org_id, title, scope, created_by) VALUES ($1,$2,$3,$4) RETURNING id`,
		orgID, title, scope, userID,
	).Scan(&id)
	return id, err
}

// ListPortalRequests devuelve SOLO las solicitudes de la organización dada
// (aislamiento entre tenants).
func (d *DB) ListPortalRequests(orgID int64) ([]PortalRequest, error) {
	rows, err := d.Query(
		`SELECT id, org_id, title, scope, status, COALESCE(created_by,0), created_at
         FROM portal_requests WHERE org_id = $1 ORDER BY created_at DESC LIMIT 500`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PortalRequest{}
	for rows.Next() {
		var r PortalRequest
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Title, &r.Scope, &r.Status, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
