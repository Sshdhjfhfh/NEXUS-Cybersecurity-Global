// Two-factor authentication (TOTP, RFC 6238) for the console. The shared secret
// is stored encrypted at rest (AES-256-GCM, key derived from the JWT signing
// key) so a database leak alone does not reveal the 2FA seed.
package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

const (
	mfaEnabledKey = "auth.mfa_enabled"
	mfaSecretKey  = "auth.mfa_secret"
	totpDigits    = 6
	totpPeriod    = 30
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// --- TOTP (RFC 6238) ---------------------------------------------------------

func newTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return b32.EncodeToString(b)
}

func totpAt(secret string, t time.Time) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(t.Unix())/totpPeriod)
	h := hmac.New(sha1.New, key)
	h.Write(msg[:])
	sum := h.Sum(nil)
	o := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[o]&0x7f)<<24 | uint32(sum[o+1])<<16 | uint32(sum[o+2])<<8 | uint32(sum[o+3])) % 1_000_000
	return fmt.Sprintf("%06d", code), nil
}

// verifyTOTP checks the code against the current and adjacent 30s windows to
// tolerate clock skew, in constant time.
func verifyTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}
	now := time.Now()
	for _, skew := range []time.Duration{0, -totpPeriod * time.Second, totpPeriod * time.Second} {
		if c, err := totpAt(secret, now.Add(skew)); err == nil &&
			subtle.ConstantTimeCompare([]byte(c), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func otpauthURI(secret string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", "NEXUS Cybersecurity")
	v.Set("algorithm", "SHA1")
	v.Set("digits", strconv.Itoa(totpDigits))
	v.Set("period", strconv.Itoa(totpPeriod))
	return "otpauth://totp/" + url.PathEscape("NEXUS:ARTEX") + "?" + v.Encode()
}

// --- secret encryption at rest ----------------------------------------------

func (s *Server) mfaCipher() (cipher.AEAD, error) {
	sum := sha256.Sum256(s.jwtKey)
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Server) encryptSecret(plain string) (string, error) {
	gcm, err := s.mfaCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (s *Server) decryptSecret(enc string) (string, error) {
	gcm, err := s.mfaCipher()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// --- state helpers -----------------------------------------------------------

func (s *Server) mfaEnabled(pg *db.DB) bool {
	v, ok, err := pg.GetSetting(mfaEnabledKey)
	return err == nil && ok && v == "1"
}

// mfaCheckPasses returns true when MFA is disabled, or enabled and the supplied
// code is valid. Used during login.
func (s *Server) mfaCheckPasses(pg *db.DB, code string) bool {
	if !s.mfaEnabled(pg) {
		return true
	}
	enc, ok, err := pg.GetSetting(mfaSecretKey)
	if err != nil || !ok || enc == "" {
		return false
	}
	secret, err := s.decryptSecret(enc)
	if err != nil {
		return false
	}
	return verifyTOTP(secret, code)
}

// --- endpoints (JWT validated inline; these sit under /api/auth/*) -----------

func (s *Server) mfaAuthed(w http.ResponseWriter, r *http.Request) bool {
	if !verifyJWT(extractToken(r), s.jwtKey) {
		writeErr(w, 401, "未授权")
		return false
	}
	return true
}

// GET /api/auth/mfa/status
func (s *Server) mfaStatus(w http.ResponseWriter, r *http.Request) {
	if !s.mfaAuthed(w, r) {
		return
	}
	pg := s.pg(w)
	if pg == nil {
		return
	}
	writeJSON(w, 200, map[string]any{"enabled": s.mfaEnabled(pg)})
}

// POST /api/auth/mfa/setup — generates a new secret (pending until enabled).
func (s *Server) mfaSetup(w http.ResponseWriter, r *http.Request) {
	if !s.mfaAuthed(w, r) {
		return
	}
	pg := s.pg(w)
	if pg == nil {
		return
	}
	secret := newTOTPSecret()
	enc, err := s.encryptSecret(secret)
	if err != nil {
		writeErr(w, 500, "no se pudo generar el secreto")
		return
	}
	if err := pg.SetSetting(mfaSecretKey, enc); err != nil {
		writeErr(w, 500, "no se pudo guardar el secreto")
		return
	}
	writeJSON(w, 200, map[string]any{"secret": secret, "otpauth": otpauthURI(secret)})
}

// POST /api/auth/mfa/enable {code} — verifies the pending secret and turns MFA on.
func (s *Server) mfaEnable(w http.ResponseWriter, r *http.Request) {
	if !s.mfaAuthed(w, r) {
		return
	}
	pg := s.pg(w)
	if pg == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "请求格式错误")
		return
	}
	enc, ok, err := pg.GetSetting(mfaSecretKey)
	if err != nil || !ok || enc == "" {
		writeErr(w, 400, "primero ejecuta la configuración")
		return
	}
	secret, err := s.decryptSecret(enc)
	if err != nil {
		writeErr(w, 500, "secreto inválido")
		return
	}
	if !verifyTOTP(secret, req.Code) {
		writeErr(w, 400, "código inválido")
		return
	}
	if err := pg.SetSetting(mfaEnabledKey, "1"); err != nil {
		writeErr(w, 500, "no se pudo activar")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/auth/mfa/disable {code} — turns MFA off (requires a valid code).
func (s *Server) mfaDisable(w http.ResponseWriter, r *http.Request) {
	if !s.mfaAuthed(w, r) {
		return
	}
	pg := s.pg(w)
	if pg == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "请求格式错误")
		return
	}
	if s.mfaEnabled(pg) && !s.mfaCheckPasses(pg, req.Code) {
		writeErr(w, 400, "código inválido")
		return
	}
	_ = pg.SetSetting(mfaEnabledKey, "")
	_ = pg.SetSetting(mfaSecretKey, "")
	writeJSON(w, 200, map[string]any{"ok": true})
}
