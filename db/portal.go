// Nexus IA v6.0 — Portal de clientes multi-tenant (código original de NEXUS).
// Cada organización (tenant) y sus datos están aislados por org_id; todas las
// lecturas del portal filtran SIEMPRE por org_id para que un cliente nunca vea
// datos de otro.
package db

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrEmailTaken se devuelve cuando el email ya está registrado.
var ErrEmailTaken = errors.New("email already registered")

type Org struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Suspended bool      `json:"suspended"`
	CreatedAt time.Time `json:"created_at"`
}

type PortalUser struct {
	ID            int64  `json:"id"`
	OrgID         int64  `json:"org_id"`
	Email         string `json:"email"`
	PasswordHash  string `json:"-"`
	Role          string `json:"role"`
	MFASecret     string `json:"-"` // encrypted at rest; empty when no secret set
	MFAEnabled    bool   `json:"mfa_enabled"`
	EmailVerified bool   `json:"email_verified"`
	Disabled      bool   `json:"disabled"`
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
-- Platform operator can suspend an organization (blocks all portal access).
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS suspended BOOLEAN NOT NULL DEFAULT false;
-- Per-user TOTP 2FA. Secret stored encrypted at rest (AES-256-GCM); empty = none.
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS mfa_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT false;
-- Email verification state.
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;
-- Per-user deactivation + session revocation. A disabled user is blocked at login
-- and on every request; tokens issued at/before sessions_revoked_at are rejected.
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS disabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS sessions_revoked_at TIMESTAMPTZ;
-- Per-user UI preferences, kept server-side so they follow the user across
-- devices: the dashboard perspective ('executive' | 'technical' | '' = default
-- by role) and the interface language ('es' | 'en' | '' = browser default).
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS view_profile TEXT NOT NULL DEFAULT '';
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS lang TEXT NOT NULL DEFAULT '';
-- Client-side remediation tracking for a released finding. The engine's finding
-- row is never modified from the portal: "fixed" is only ever set by a NEXUS
-- retest. client_state is what the client's team reports.
CREATE TABLE IF NOT EXISTS portal_remediation (
    org_id              BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    finding_id          BIGINT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    assignee_id         BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    due_date            DATE,
    client_state        TEXT NOT NULL DEFAULT 'open'
                        CHECK (client_state IN ('open','in_progress','ready_for_retest','risk_accepted')),
    retest_requested_at TIMESTAMPTZ,
    retest_requested_by BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    retest_note         TEXT NOT NULL DEFAULT '',
    risk_reason         TEXT NOT NULL DEFAULT '',
    risk_expires_at     DATE,
    risk_accepted_by    BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    risk_accepted_at    TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, finding_id)
);
CREATE INDEX IF NOT EXISTS idx_portal_remediation_state ON portal_remediation(client_state);
-- Per-finding remediation timeline (comments + changes), shown to the client.
CREATE TABLE IF NOT EXISTS portal_remediation_events (
    id          BIGSERIAL PRIMARY KEY,
    org_id      BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    finding_id  BIGINT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    actor       TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_remediation_events ON portal_remediation_events(org_id, finding_id, id);
-- One-time tokens for password reset and email verification. Only the SHA-256 of
-- the token is stored; the plaintext lives only in the email sent to the user.
CREATE TABLE IF NOT EXISTS portal_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_tokens_hash ON portal_tokens(token_hash);
-- Link a promoted request to the engine task an operator created from it.
ALTER TABLE portal_requests ADD COLUMN IF NOT EXISTS engine_task_id BIGINT;
-- Operator gate: results are shown to the client only once explicitly released.
ALTER TABLE portal_requests ADD COLUMN IF NOT EXISTS results_released BOOLEAN NOT NULL DEFAULT false;
CREATE TABLE IF NOT EXISTS portal_invites (
    id          BIGSERIAL PRIMARY KEY,
    org_id      BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',
    token       TEXT NOT NULL UNIQUE,
    created_by  BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    accepted    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_portal_invites_org ON portal_invites(org_id);
-- Tamper-evident audit trail (hash-chained). No FK on org_id so entries survive
-- org deletion (audit must be retained for compliance).
CREATE TABLE IF NOT EXISTS portal_audit (
    id          BIGSERIAL PRIMARY KEY,
    org_id      BIGINT,
    actor       TEXT NOT NULL DEFAULT '',
    actor_type  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    target      TEXT NOT NULL DEFAULT '',
    detail      TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    prev_hash   TEXT NOT NULL DEFAULT '',
    hash        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_portal_audit_org ON portal_audit(org_id);
CREATE INDEX IF NOT EXISTS idx_portal_audit_id_desc ON portal_audit(id DESC);
-- Email notifications. notify_email is the user's opt-out for non-security mail
-- (password reset / email verification are always sent).
ALTER TABLE portal_users ADD COLUMN IF NOT EXISTS notify_email BOOLEAN NOT NULL DEFAULT true;
-- Ledger of notifications already claimed, so an event is mailed at most once
-- (claimed with INSERT ... ON CONFLICT DO NOTHING before anything is queued).
CREATE TABLE IF NOT EXISTS portal_notifications (
    id         BIGSERIAL PRIMARY KEY,
    org_id     BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    ref        TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, kind, ref)
);
-- Outbox for notification mail: delivered by a background worker with retries,
-- so a slow or failing SMTP server never blocks a request. Never holds tokens.
CREATE TABLE IF NOT EXISTS portal_mail_outbox (
    id         BIGSERIAL PRIMARY KEY,
    org_id     BIGINT REFERENCES orgs(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL DEFAULT '',
    to_addr    TEXT NOT NULL,
    subject    TEXT NOT NULL,
    text_body  TEXT NOT NULL,
    html_body  TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
    attempts   INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_portal_mail_outbox_pending ON portal_mail_outbox(next_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_portal_mail_outbox_id_desc ON portal_mail_outbox(id DESC);
-- Targets an org wants tested. Nothing is tested until the org proves control of
-- it (DNS TXT or a file on the web server) or a NEXUS operator approves it on
-- documentary evidence (e.g. a signed authorization letter). Verification lasts
-- a limited time and must then be renewed.
CREATE TABLE IF NOT EXISTS portal_targets (
    id            BIGSERIAL PRIMARY KEY,
    org_id        BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN ('domain','ip')),
    value         TEXT NOT NULL,
    token         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','verified','rejected')),
    method        TEXT NOT NULL DEFAULT '' CHECK (method IN ('','dns','http','manual')),
    verified_at   TIMESTAMPTZ,
    verified_by   TEXT NOT NULL DEFAULT '',
    note          TEXT NOT NULL DEFAULT '',
    last_check_at TIMESTAMPTZ,
    last_error    TEXT NOT NULL DEFAULT '',
    created_by    BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, value)
);
CREATE INDEX IF NOT EXISTS idx_portal_targets_status ON portal_targets(status);
-- Which targets an evaluation request covers.
CREATE TABLE IF NOT EXISTS portal_request_targets (
    request_id BIGINT NOT NULL REFERENCES portal_requests(id) ON DELETE CASCADE,
    target_id  BIGINT NOT NULL REFERENCES portal_targets(id) ON DELETE CASCADE,
    PRIMARY KEY (request_id, target_id)
);
-- Rules of Engagement accepted for a request by an org admin: who signed, when,
-- from where, the testing window, emergency contact, exclusions, and the exact
-- version + SHA-256 of the terms shown to them.
CREATE TABLE IF NOT EXISTS portal_roe (
    request_id        BIGINT PRIMARY KEY REFERENCES portal_requests(id) ON DELETE CASCADE,
    org_id            BIGINT NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    accepted_by       BIGINT REFERENCES portal_users(id) ON DELETE SET NULL,
    accepted_email    TEXT NOT NULL,
    signer_name       TEXT NOT NULL,
    signer_title      TEXT NOT NULL,
    window_start      DATE NOT NULL,
    window_end        DATE NOT NULL,
    emergency_contact TEXT NOT NULL,
    exclusions        TEXT NOT NULL DEFAULT '',
    roe_version       TEXT NOT NULL,
    roe_hash          TEXT NOT NULL,
    ip                TEXT NOT NULL DEFAULT '',
    accepted_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// portalSchemaDone records the connection pools whose portal schema is already
// in place. The DDL used to run on every request; ALTER TABLE ... IF NOT EXISTS
// takes an exclusive lock even when it changes nothing, so concurrent requests
// could deadlock (sporadic 503s). Now it runs once per pool and process, and is
// retried only until it succeeds.
var (
	portalSchemaMu   sync.Mutex
	portalSchemaDone = map[*sql.DB]bool{}
)

// EnsurePortalTables crea las tablas del portal si no existen (aditivo, reversible).
func (d *DB) EnsurePortalTables() error {
	portalSchemaMu.Lock()
	defer portalSchemaMu.Unlock()
	if portalSchemaDone[d.DB] {
		return nil
	}
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
	if err := tx.Commit(); err != nil {
		return err
	}
	portalSchemaDone[d.DB] = true
	return nil
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
		`SELECT id, org_id, email, password_hash, role, mfa_secret, mfa_enabled, email_verified, disabled FROM portal_users WHERE email = $1`, email,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.MFASecret, &u.MFAEnabled, &u.EmailVerified, &u.Disabled)
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
	err := d.QueryRow(`SELECT id, name, suspended, created_at FROM orgs WHERE id = $1`, id).
		Scan(&o.ID, &o.Name, &o.Suspended, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// OrgSummary es una organización con contadores agregados, para la consola de
// plataforma del operador.
type OrgSummary struct {
	Org
	MemberCount  int `json:"member_count"`
	RequestCount int `json:"request_count"`
}

// AdminListOrgs lista TODAS las organizaciones con sus contadores (operador).
func (d *DB) AdminListOrgs() ([]OrgSummary, error) {
	rows, err := d.Query(
		`SELECT o.id, o.name, o.suspended, o.created_at,
                (SELECT COUNT(*) FROM portal_users u WHERE u.org_id = o.id)    AS members,
                (SELECT COUNT(*) FROM portal_requests r WHERE r.org_id = o.id) AS requests
         FROM orgs o ORDER BY o.created_at DESC LIMIT 1000`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrgSummary{}
	for rows.Next() {
		var s OrgSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Suspended, &s.CreatedAt, &s.MemberCount, &s.RequestCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SetOrgSuspended suspende o reactiva una organización.
func (d *DB) SetOrgSuspended(id int64, suspended bool) error {
	res, err := d.Exec(`UPDATE orgs SET suspended = $1 WHERE id = $2`, suspended, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// OrgSuspended indica si la organización está suspendida. Se usa para bloquear el
// acceso del portal en cada petición autenticada.
func (d *DB) OrgSuspended(id int64) (bool, error) {
	var suspended bool
	err := d.QueryRow(`SELECT suspended FROM orgs WHERE id = $1`, id).Scan(&suspended)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return suspended, err
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

// Member es un usuario del portal visible en la gestión de equipo (sin hash).
type Member struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
}

// PortalInvite es una invitación pendiente/aceptada a una organización.
type PortalInvite struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Token     string    `json:"token,omitempty"`
	Accepted  bool      `json:"accepted"`
	CreatedAt time.Time `json:"created_at"`
}

// GetPortalUserByID devuelve el usuario por id (o nil si no existe). Se usa para
// resolver el rol actual desde la base de datos en cada petición (no desde el JWT,
// que podría quedar obsoleto si cambia el rol).
func (d *DB) GetPortalUserByID(id int64) (*PortalUser, error) {
	var u PortalUser
	err := d.QueryRow(
		`SELECT id, org_id, email, password_hash, role, mfa_secret, mfa_enabled, email_verified, disabled FROM portal_users WHERE id = $1`, id,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.MFASecret, &u.MFAEnabled, &u.EmailVerified, &u.Disabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListMembers devuelve SOLO los miembros de la organización dada (aislamiento).
func (d *DB) ListMembers(orgID int64) ([]Member, error) {
	rows, err := d.Query(
		`SELECT id, org_id, email, role, disabled, created_at FROM portal_users
         WHERE org_id = $1 ORDER BY created_at ASC LIMIT 500`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.OrgID, &m.Email, &m.Role, &m.Disabled, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountOwners cuenta los propietarios activos de una organización. Se usa para
// impedir que una organización se quede sin ningún owner.
func (d *DB) CountOwners(orgID int64) (int, error) {
	var n int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM portal_users WHERE org_id = $1 AND role = 'owner'`, orgID,
	).Scan(&n)
	return n, err
}

// CountActiveOwners cuenta los propietarios NO desactivados. Impide dejar la
// organización sin ningún owner que pueda acceder.
func (d *DB) CountActiveOwners(orgID int64) (int, error) {
	var n int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM portal_users WHERE org_id = $1 AND role = 'owner' AND disabled = false`, orgID,
	).Scan(&n)
	return n, err
}

// PortalAuthInfo es el estado que autoriza cada petición del portal.
type PortalAuthInfo struct {
	OrgID     int64
	Role      string
	Disabled  bool
	RevokedAt *time.Time
	Suspended bool
}

// PortalAuthState resuelve, en una sola consulta, el estado necesario para
// autorizar cada petición del portal: si el usuario existe (nil si no), su rol
// actual, si está desactivado, cuándo se revocaron sus sesiones, y si su
// organización está suspendida.
func (d *DB) PortalAuthState(userID int64) (*PortalAuthInfo, error) {
	var a PortalAuthInfo
	var rev sql.NullTime
	err := d.QueryRow(
		`SELECT u.org_id, u.role, u.disabled, u.sessions_revoked_at, o.suspended
         FROM portal_users u JOIN orgs o ON o.id = u.org_id WHERE u.id = $1`, userID,
	).Scan(&a.OrgID, &a.Role, &a.Disabled, &rev, &a.Suspended)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rev.Valid {
		t := rev.Time
		a.RevokedAt = &t
	}
	return &a, nil
}

// SetPortalUserDisabled desactiva o reactiva un usuario (acotado a su org).
func (d *DB) SetPortalUserDisabled(orgID, userID int64, disabled bool) error {
	res, err := d.Exec(
		`UPDATE portal_users SET disabled = $1 WHERE id = $2 AND org_id = $3`, disabled, userID, orgID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RevokePortalUserSessions invalida todos los tokens activos de un usuario (los
// emitidos hasta ahora) marcando sessions_revoked_at = now. Acotado a la org.
func (d *DB) RevokePortalUserSessions(orgID, userID int64) error {
	res, err := d.Exec(
		`UPDATE portal_users SET sessions_revoked_at = now() WHERE id = $1 AND org_id = $2`, userID, orgID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RevokePortalUserSessionsByID invalida las sesiones de un usuario sin acotar a
// una org (se usa tras un restablecimiento de contraseña).
func (d *DB) RevokePortalUserSessionsByID(userID int64) error {
	_, err := d.Exec(`UPDATE portal_users SET sessions_revoked_at = now() WHERE id = $1`, userID)
	return err
}

// UpdateMemberRole cambia el rol de un miembro, siempre acotado a su organización.
func (d *DB) UpdateMemberRole(orgID, userID int64, role string) error {
	res, err := d.Exec(
		`UPDATE portal_users SET role = $1 WHERE id = $2 AND org_id = $3`, role, userID, orgID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteMember elimina un miembro de la organización (acotado a org_id).
func (d *DB) DeleteMember(orgID, userID int64) error {
	res, err := d.Exec(`DELETE FROM portal_users WHERE id = $1 AND org_id = $2`, userID, orgID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CreateInvite crea una invitación pendiente para la organización dada.
func (d *DB) CreateInvite(orgID, createdBy int64, email, role, token string) (int64, error) {
	var id int64
	err := d.QueryRow(
		`INSERT INTO portal_invites(org_id, email, role, token, created_by) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		orgID, email, role, token, createdBy,
	).Scan(&id)
	return id, err
}

// GetInviteByToken resuelve una invitación por su token (o nil si no existe).
func (d *DB) GetInviteByToken(token string) (*PortalInvite, error) {
	var inv PortalInvite
	err := d.QueryRow(
		`SELECT id, org_id, email, role, token, accepted, created_at FROM portal_invites WHERE token = $1`, token,
	).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token, &inv.Accepted, &inv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListInvites devuelve SOLO las invitaciones pendientes de la organización dada.
func (d *DB) ListInvites(orgID int64) ([]PortalInvite, error) {
	rows, err := d.Query(
		`SELECT id, org_id, email, role, accepted, created_at FROM portal_invites
         WHERE org_id = $1 AND accepted = false ORDER BY created_at DESC LIMIT 500`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PortalInvite{}
	for rows.Next() {
		var inv PortalInvite
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Accepted, &inv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// DeleteInvite revoca una invitación (acotado a org_id).
func (d *DB) DeleteInvite(orgID, id int64) error {
	res, err := d.Exec(`DELETE FROM portal_invites WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetPortalMFASecret guarda un secreto TOTP (cifrado) pendiente de activar; deja
// mfa_enabled en false hasta que el usuario confirme con un código válido.
func (d *DB) SetPortalMFASecret(userID int64, secretEnc string) error {
	_, err := d.Exec(
		`UPDATE portal_users SET mfa_secret = $1, mfa_enabled = false WHERE id = $2`, secretEnc, userID,
	)
	return err
}

// EnablePortalMFA activa el 2FA (el secreto ya está guardado).
func (d *DB) EnablePortalMFA(userID int64) error {
	res, err := d.Exec(`UPDATE portal_users SET mfa_enabled = true WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DisablePortalMFA desactiva el 2FA y borra el secreto.
func (d *DB) DisablePortalMFA(userID int64) error {
	_, err := d.Exec(`UPDATE portal_users SET mfa_secret = '', mfa_enabled = false WHERE id = $1`, userID)
	return err
}

// AdminPortalRequest es una solicitud con el nombre de su organización y el
// posible vínculo a una tarea del motor. Solo la ve el operador de la consola
// (requireAuth), nunca el cliente del portal.
type AdminPortalRequest struct {
	ID              int64     `json:"id"`
	OrgID           int64     `json:"org_id"`
	OrgName         string    `json:"org_name"`
	Title           string    `json:"title"`
	Scope           string    `json:"scope"`
	Status          string    `json:"status"`
	EngineTaskID    *int64    `json:"engine_task_id,omitempty"`
	ResultsReleased bool      `json:"results_released"`
	CreatedAt       time.Time `json:"created_at"`
}

func scanAdminRequest(row interface{ Scan(...any) error }) (AdminPortalRequest, error) {
	var r AdminPortalRequest
	var taskID sql.NullInt64
	if err := row.Scan(&r.ID, &r.OrgID, &r.OrgName, &r.Title, &r.Scope, &r.Status, &taskID, &r.ResultsReleased, &r.CreatedAt); err != nil {
		return r, err
	}
	if taskID.Valid {
		r.EngineTaskID = &taskID.Int64
	}
	return r, nil
}

const adminRequestCols = `r.id, r.org_id, o.name, r.title, r.scope, r.status, r.engine_task_id, r.results_released, r.created_at`

// AdminListPortalRequests devuelve TODAS las solicitudes (todas las orgs) para
// el operador, con el nombre de la organización.
func (d *DB) AdminListPortalRequests() ([]AdminPortalRequest, error) {
	rows, err := d.Query(
		`SELECT ` + adminRequestCols + ` FROM portal_requests r
         JOIN orgs o ON o.id = r.org_id ORDER BY r.created_at DESC LIMIT 1000`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminPortalRequest{}
	for rows.Next() {
		r, err := scanAdminRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetPortalRequest devuelve una solicitud (cualquier org) por id, o nil.
func (d *DB) GetPortalRequest(id int64) (*AdminPortalRequest, error) {
	r, err := scanAdminRequest(d.QueryRow(
		`SELECT `+adminRequestCols+` FROM portal_requests r
         JOIN orgs o ON o.id = r.org_id WHERE r.id = $1`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// LinkRequestToTask vincula la solicitud a una tarea del motor y la pasa a
// 'scheduled'. Solo enlaza si aún no tenía tarea (evita duplicar).
func (d *DB) LinkRequestToTask(reqID, taskID int64) error {
	res, err := d.Exec(
		`UPDATE portal_requests SET engine_task_id = $1, status = 'scheduled'
         WHERE id = $2 AND engine_task_id IS NULL`, taskID, reqID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdatePortalRequestStatus cambia el estado de una solicitud (operador).
func (d *DB) UpdatePortalRequestStatus(id int64, status string) error {
	res, err := d.Exec(`UPDATE portal_requests SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AcceptInvite crea el usuario de la invitación y la marca aceptada en una
// transacción. Devuelve ErrEmailTaken si el email ya está registrado.
func (d *DB) AcceptInvite(inviteID, orgID int64, email, role, passwordHash string) (userID int64, err error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	err = tx.QueryRow(
		`INSERT INTO portal_users(org_id, email, password_hash, role) VALUES ($1,$2,$3,$4) RETURNING id`,
		orgID, email, passwordHash, role,
	).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	if _, err = tx.Exec(`UPDATE portal_invites SET accepted = true WHERE id = $1`, inviteID); err != nil {
		return 0, err
	}
	return userID, tx.Commit()
}
