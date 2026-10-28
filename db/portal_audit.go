// Tamper-evident audit trail for the portal/operator domain. Each entry stores a
// SHA-256 hash over the previous entry's hash plus this entry's fields, forming a
// chain: altering or deleting any past row breaks every hash after it, which
// VerifyAuditChain detects. Appends are serialized (auditMu) so the chain stays
// well-ordered in this single-process backend. Entries are never updated or
// deleted in normal operation.
package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AuditEntry is one immutable, hash-chained audit record.
type AuditEntry struct {
	ID        int64     `json:"id"`
	OrgID     *int64    `json:"org_id,omitempty"`
	Actor     string    `json:"actor"`
	ActorType string    `json:"actor_type"` // portal_user | operator | anonymous
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail,omitempty"`
	IP        string    `json:"ip"`
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

// auditMu serializes chain appends so prev_hash always refers to the true tail.
var auditMu sync.Mutex

func orgIDStr(orgID *int64) string {
	if orgID == nil {
		return "0"
	}
	return strconv.FormatInt(*orgID, 10)
}

// auditHash binds an entry to its predecessor. The canonical form is a fixed,
// order-stable join of every field that matters; changing any of them (or the
// prev hash) changes the result.
func auditHash(prev string, e AuditEntry) string {
	// Hash the timestamp as integer microseconds: Postgres timestamptz keeps only
	// microseconds and may return a different zone, so a formatted RFC3339 string
	// would not survive the DB roundtrip (and the chain would never verify).
	// UnixMicro is zone-independent and exact at the stored precision.
	canon := strings.Join([]string{
		prev,
		orgIDStr(e.OrgID),
		e.Actor,
		e.ActorType,
		e.Action,
		e.Target,
		e.Detail,
		e.IP,
		strconv.FormatInt(e.CreatedAt.UnixMicro(), 10),
	}, "|")
	sum := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(sum[:])
}

// AppendAudit adds one entry to the chain and returns it with its hashes set.
func (d *DB) AppendAudit(e AuditEntry) (AuditEntry, error) {
	auditMu.Lock()
	defer auditMu.Unlock()

	tx, err := d.Begin()
	if err != nil {
		return e, err
	}
	defer tx.Rollback() //nolint:errcheck

	var prev string
	err = tx.QueryRow(`SELECT hash FROM portal_audit ORDER BY id DESC LIMIT 1`).Scan(&prev)
	if errors.Is(err, sql.ErrNoRows) {
		prev = "" // genesis
	} else if err != nil {
		return e, err
	}

	// Truncate to microseconds so the stored value and the hashed value are
	// identical after the Postgres roundtrip (timestamptz = microsecond precision).
	e.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	e.PrevHash = prev
	e.Hash = auditHash(prev, e)
	err = tx.QueryRow(
		`INSERT INTO portal_audit(org_id, actor, actor_type, action, target, detail, ip, prev_hash, hash, created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		e.OrgID, e.Actor, e.ActorType, e.Action, e.Target, e.Detail, e.IP, e.PrevHash, e.Hash, e.CreatedAt,
	).Scan(&e.ID)
	if err != nil {
		return e, err
	}
	return e, tx.Commit()
}

// ListAudit returns audit entries newest-first. When orgID is non-nil it is
// scoped to that organization (tenant self-view); otherwise it returns all
// (operator view).
func (d *DB) ListAudit(orgID *int64, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	var rows *sql.Rows
	var err error
	if orgID != nil {
		rows, err = d.Query(
			`SELECT id, org_id, actor, actor_type, action, target, detail, ip, prev_hash, hash, created_at
             FROM portal_audit WHERE org_id = $1 ORDER BY id DESC LIMIT $2`, *orgID, limit,
		)
	} else {
		rows, err = d.Query(
			`SELECT id, org_id, actor, actor_type, action, target, detail, ip, prev_hash, hash, created_at
             FROM portal_audit ORDER BY id DESC LIMIT $1`, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAudit(rows)
}

func scanAudit(rows *sql.Rows) ([]AuditEntry, error) {
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var org sql.NullInt64
		if err := rows.Scan(&e.ID, &org, &e.Actor, &e.ActorType, &e.Action, &e.Target,
			&e.Detail, &e.IP, &e.PrevHash, &e.Hash, &e.CreatedAt); err != nil {
			return nil, err
		}
		if org.Valid {
			e.OrgID = &org.Int64
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AuditVerification reports whether the chain is intact.
type AuditVerification struct {
	OK       bool   `json:"ok"`
	Count    int    `json:"count"`
	BrokenAt *int64 `json:"broken_at,omitempty"` // id of the first tampered/missing-link entry
	Reason   string `json:"reason,omitempty"`
}

// VerifyAuditChain re-walks the whole chain in order and recomputes every hash,
// confirming each entry links to the previous one. Any modification, deletion or
// reordering of a past entry is detected.
func (d *DB) VerifyAuditChain() (AuditVerification, error) {
	rows, err := d.Query(
		`SELECT id, org_id, actor, actor_type, action, target, detail, ip, prev_hash, hash, created_at
         FROM portal_audit ORDER BY id ASC`,
	)
	if err != nil {
		return AuditVerification{}, err
	}
	defer rows.Close()

	prev := ""
	count := 0
	for rows.Next() {
		var e AuditEntry
		var org sql.NullInt64
		if err := rows.Scan(&e.ID, &org, &e.Actor, &e.ActorType, &e.Action, &e.Target,
			&e.Detail, &e.IP, &e.PrevHash, &e.Hash, &e.CreatedAt); err != nil {
			return AuditVerification{}, err
		}
		if org.Valid {
			e.OrgID = &org.Int64
		}
		count++
		if e.PrevHash != prev {
			id := e.ID
			return AuditVerification{OK: false, Count: count, BrokenAt: &id, Reason: "prev_hash no enlaza con la entrada anterior"}, nil
		}
		if auditHash(prev, e) != e.Hash {
			id := e.ID
			return AuditVerification{OK: false, Count: count, BrokenAt: &id, Reason: "el hash de la entrada no coincide (datos alterados)"}, nil
		}
		prev = e.Hash
	}
	if err := rows.Err(); err != nil {
		return AuditVerification{}, err
	}
	return AuditVerification{OK: true, Count: count}, nil
}
