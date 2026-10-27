// Nexus IA v6.0 — captación de solicitudes comerciales (leads) de la web pública.
// Código original de NEXUS (no derivado de ARTEX).
package db

import "time"

// Lead is one commercial contact/demo request captured from the public website.
type Lead struct {
	ID           int64     `json:"id"`
	Organization string    `json:"organization"`
	ContactName  string    `json:"contact_name"`
	Email        string    `json:"email"`
	Country      string    `json:"country"`
	Service      string    `json:"service"`
	Message      string    `json:"message"`
	Status       string    `json:"status"` // new | contacted | qualified | closed
	Consent      bool      `json:"consent"`
	SourceIP     string    `json:"source_ip,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

const leadsSchema = `
CREATE TABLE IF NOT EXISTS leads (
    id            BIGSERIAL PRIMARY KEY,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    organization  TEXT NOT NULL,
    contact_name  TEXT NOT NULL,
    email         TEXT NOT NULL,
    country       TEXT NOT NULL DEFAULT '',
    service       TEXT NOT NULL DEFAULT '',
    message       TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'new',
    consent       BOOLEAN NOT NULL DEFAULT false,
    source_ip     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_leads_created ON leads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_leads_status  ON leads(status);
`

// EnsureLeadsTable creates the leads table if it does not exist (reversible: a
// plain additive table, no change to existing data).
func (d *DB) EnsureLeadsTable() error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := coordinateWithSchemaMigration(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(leadsSchema); err != nil {
		return err
	}
	return tx.Commit()
}

// InsertLead stores a new commercial request and returns its id.
func (d *DB) InsertLead(l *Lead) (int64, error) {
	var id int64
	err := d.QueryRow(`
INSERT INTO leads(organization, contact_name, email, country, service, message, status, consent, source_ip)
VALUES ($1,$2,$3,$4,$5,$6,COALESCE(NULLIF($7,''),'new'),$8,$9)
RETURNING id`,
		l.Organization, l.ContactName, l.Email, l.Country, l.Service, l.Message,
		l.Status, l.Consent, l.SourceIP).Scan(&id)
	return id, err
}

// CountLeadsFromIPSince counts leads created by one source IP since a cutoff —
// the input to the per-IP submission rate limit.
func (d *DB) CountLeadsFromIPSince(ip string, since time.Time) (int, error) {
	var n int
	err := d.QueryRow(`SELECT COUNT(*) FROM leads WHERE source_ip = $1 AND created_at >= $2`, ip, since).Scan(&n)
	return n, err
}

// ListLeads returns the most recent leads, newest first (admin view). limit is
// clamped to [1, 500].
func (d *DB) ListLeads(limit int) ([]Lead, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := d.Query(`
SELECT id, created_at, organization, contact_name, email, country, service, message, status, consent, source_ip
FROM leads ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Lead{}
	for rows.Next() {
		var l Lead
		if err := rows.Scan(&l.ID, &l.CreatedAt, &l.Organization, &l.ContactName, &l.Email,
			&l.Country, &l.Service, &l.Message, &l.Status, &l.Consent, &l.SourceIP); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// LeadStatuses are the allowed workflow states for a lead.
var LeadStatuses = map[string]bool{"new": true, "contacted": true, "qualified": true, "closed": true}

// UpdateLeadStatus moves a lead to a new workflow state. Returns false if no lead
// with that id exists. Reversible: it only changes the status column.
func (d *DB) UpdateLeadStatus(id int64, status string) (bool, error) {
	res, err := d.Exec(`UPDATE leads SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// CountLeads returns the total number of leads (admin KPI).
func (d *DB) CountLeads() (int, error) {
	var n int
	err := d.QueryRow(`SELECT COUNT(*) FROM leads`).Scan(&n)
	return n, err
}
