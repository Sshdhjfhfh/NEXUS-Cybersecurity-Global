package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// PortalSSO is an organization's OpenID Connect single sign-on configuration.
// The client secret is stored encrypted (AES-256-GCM) and never returned.
type PortalSSO struct {
	OrgID         int64      `json:"-"`
	Issuer        string     `json:"issuer"`
	ClientID      string     `json:"client_id"`
	SecretEnc     string     `json:"-"`
	EmailDomains  []string   `json:"email_domains"`
	Enabled       bool       `json:"enabled"`
	Enforce       bool       `json:"enforce"`        // password sign-in refused (owners keep it as break-glass)
	AutoProvision bool       `json:"auto_provision"` // create accounts on first SSO sign-in
	DefaultRole   string     `json:"default_role"`   // role for auto-provisioned accounts
	UpdatedAt     *time.Time `json:"updated_at"`
	UpdatedBy     string     `json:"updated_by"`
}

// ErrSSODomainTaken: another organization already uses that email domain for SSO.
var ErrSSODomainTaken = errors.New("sso domain taken")

const ssoCols = `org_id, issuer, client_id, client_secret_enc, email_domains, enabled, enforce,
                 auto_provision, default_role, updated_at, updated_by`

func scanSSO(row interface{ Scan(...any) error }) (*PortalSSO, error) {
	var c PortalSSO
	var domains string
	var upd sql.NullTime
	err := row.Scan(&c.OrgID, &c.Issuer, &c.ClientID, &c.SecretEnc, &domains, &c.Enabled, &c.Enforce,
		&c.AutoProvision, &c.DefaultRole, &upd, &c.UpdatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.EmailDomains = splitAllowlist(domains)
	if upd.Valid {
		t := upd.Time
		c.UpdatedAt = &t
	}
	return &c, nil
}

// GetPortalSSO devuelve la configuración SSO de la org (nil si no hay).
func (d *DB) GetPortalSSO(orgID int64) (*PortalSSO, error) {
	return scanSSO(d.QueryRow(`SELECT `+ssoCols+` FROM portal_sso WHERE org_id = $1`, orgID))
}

// FindPortalSSOByDomain devuelve la configuración SSO activa que reclama el
// dominio de email indicado (nil si ninguna).
func (d *DB) FindPortalSSOByDomain(domain string) (*PortalSSO, error) {
	return scanSSO(d.QueryRow(
		`SELECT `+ssoCols+` FROM portal_sso s
         WHERE s.enabled AND EXISTS (SELECT 1 FROM portal_sso_domains x WHERE x.org_id = s.org_id AND x.domain = $1)`,
		strings.ToLower(domain)))
}

// SavePortalSSO crea o actualiza la configuración. secretEnc vacío conserva el
// secreto guardado. Los dominios son únicos entre organizaciones.
func (d *DB) SavePortalSSO(c PortalSSO) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	_, err = tx.Exec(
		`INSERT INTO portal_sso(org_id, issuer, client_id, client_secret_enc, email_domains, enabled, enforce,
                                auto_provision, default_role, updated_at, updated_by)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),$10)
         ON CONFLICT (org_id) DO UPDATE SET issuer = EXCLUDED.issuer, client_id = EXCLUDED.client_id,
             client_secret_enc = CASE WHEN EXCLUDED.client_secret_enc = '' THEN portal_sso.client_secret_enc
                                      ELSE EXCLUDED.client_secret_enc END,
             email_domains = EXCLUDED.email_domains, enabled = EXCLUDED.enabled, enforce = EXCLUDED.enforce,
             auto_provision = EXCLUDED.auto_provision, default_role = EXCLUDED.default_role,
             updated_at = now(), updated_by = EXCLUDED.updated_by`,
		c.OrgID, c.Issuer, c.ClientID, c.SecretEnc, strings.Join(c.EmailDomains, ","), c.Enabled, c.Enforce,
		c.AutoProvision, c.DefaultRole, c.UpdatedBy)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM portal_sso_domains WHERE org_id = $1`, c.OrgID); err != nil {
		return err
	}
	for _, dom := range c.EmailDomains {
		if _, err := tx.Exec(`INSERT INTO portal_sso_domains(domain, org_id) VALUES ($1,$2)`, dom, c.OrgID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrSSODomainTaken
			}
			return err
		}
	}
	return tx.Commit()
}

// DeletePortalSSO elimina la configuración SSO de la org.
func (d *DB) DeletePortalSSO(orgID int64) error {
	_, err := d.Exec(`DELETE FROM portal_sso WHERE org_id = $1`, orgID)
	return err
}

// PortalSSOEnforced indica si la org exige SSO (y lo tiene activo).
func (d *DB) PortalSSOEnforced(orgID int64) (bool, error) {
	var on bool
	err := d.QueryRow(`SELECT enabled AND enforce FROM portal_sso WHERE org_id = $1`, orgID).Scan(&on)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return on, err
}

// VerifiedDNSDomains lista los dominios de la org cuya propiedad está probada
// por DNS o aprobada por NEXUS: solo esos pueden reclamarse para SSO.
func (d *DB) VerifiedDNSDomains(orgID int64) ([]string, error) {
	rows, err := d.Query(
		`SELECT value FROM portal_targets
         WHERE org_id = $1 AND kind = 'domain' AND status = 'verified' AND method IN ('dns','manual')
           AND (verified_at IS NULL OR verified_at > now() - interval '365 days')
         ORDER BY value`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// --- one-time sign-in state (authorization request -> callback) ---

// SSOState is what a pending SSO sign-in remembers between redirect and callback.
type SSOState struct {
	OrgID       int64
	Verifier    string
	Nonce       string
	RedirectURI string
	LoginHint   string
}

// PutSSOState guarda el estado de un inicio de sesión SSO (solo el hash del state).
func (d *DB) PutSSOState(stateHash string, s SSOState) error {
	_, err := d.Exec(
		`INSERT INTO portal_sso_states(state_hash, org_id, code_verifier, nonce, redirect_uri, login_hint)
         VALUES ($1,$2,$3,$4,$5,$6)`, stateHash, s.OrgID, s.Verifier, s.Nonce, s.RedirectURI, s.LoginHint)
	if err == nil {
		_, _ = d.Exec(`DELETE FROM portal_sso_states WHERE created_at < now() - interval '1 hour'`)
	}
	return err
}

// TakeSSOState consume (un solo uso) un estado de menos de 10 minutos.
func (d *DB) TakeSSOState(stateHash string) (*SSOState, error) {
	var s SSOState
	err := d.QueryRow(
		`DELETE FROM portal_sso_states WHERE state_hash = $1 AND created_at > now() - interval '10 minutes'
         RETURNING org_id, code_verifier, nonce, redirect_uri, login_hint`, stateHash,
	).Scan(&s.OrgID, &s.Verifier, &s.Nonce, &s.RedirectURI, &s.LoginHint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CreateSSOUser da de alta un usuario aprovisionado por SSO (sin contraseña
// utilizable: el hash es un valor que bcrypt nunca acepta).
func (d *DB) CreateSSOUser(orgID int64, email, role, subject string) (int64, error) {
	var id int64
	err := d.QueryRow(
		`INSERT INTO portal_users(org_id, email, password_hash, role, email_verified, sso_subject)
         VALUES ($1,$2,'!sso',$3,true,$4) RETURNING id`, orgID, email, role, subject,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailTaken
		}
	}
	return id, err
}

// LinkSSOSubject records the IdP subject for a user on first SSO sign-in and
// marks the email verified (the IdP vouched for it). Returns false if the user
// is already linked to a different subject (account takeover attempt via a
// re-assigned email at the IdP).
func (d *DB) LinkSSOSubject(userID int64, subject string) (bool, error) {
	res, err := d.Exec(
		`UPDATE portal_users SET sso_subject = $1, email_verified = true
         WHERE id = $2 AND (sso_subject = '' OR sso_subject = $1)`, subject, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}
