// Nexus IA v6.0 — Tests del NHI Manager
package nhi_manager

import (
	"testing"
	"time"
)

// TestDilithiumKeyPair verifica la generación de claves Dilithium3.
func TestDilithiumKeyPair(t *testing.T) {
	kp, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("GenerateDilithiumKeyPair failed: %v", err)
	}
	if kp.PublicKey == nil || kp.PrivateKey == nil {
		t.Fatal("key pair has nil components")
	}

	// Firmar y verificar
	msg := []byte("nexus audit entry test")
	sig := kp.Sign(msg)
	if len(sig) == 0 {
		t.Fatal("signature is empty")
	}

	if !VerifyDilithium(kp.PublicKey, msg, sig) {
		t.Fatal("signature verification failed")
	}

	// Verificar que firma de otro mensaje falla
	if VerifyDilithium(kp.PublicKey, []byte("tampered message"), sig) {
		t.Fatal("signature verified for wrong message (SECURITY BUG)")
	}
}

// TestJWT verifica emisión y verificación de JWT.
func TestJWT(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey failed: %v", err)
	}

	nhi := &NHI{
		AgentID:        "red-agent-01",
		AgentType:      AgentTypeRed,
		DID:            NewDID(),
		AllowedActions: []ActionType{ActionScan, ActionExploit},
		AllowedAssets:  []string{"target-*.local"},
	}

	token, err := IssueJWT(key, nhi, "session-123", time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("JWT is empty")
	}

	// Verificar
	claims, err := VerifyJWT(key, token)
	if err != nil {
		t.Fatalf("VerifyJWT failed: %v", err)
	}

	if claims.AgentID != "red-agent-01" {
		t.Errorf("AgentID = %q, want red-agent-01", claims.AgentID)
	}
	if claims.AgentType != AgentTypeRed {
		t.Errorf("AgentType = %q, want RED", claims.AgentType)
	}
	if claims.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want session-123", claims.SessionID)
	}

	// Verificar permisos
	if !claims.HasPermission(ActionScan) {
		t.Error("expected SCAN permission")
	}
	if claims.HasPermission(ActionIsolateHost) {
		t.Error("did not expect ISOLATE_HOST permission")
	}
}

// TestRevocationRegistry verifica la revocación.
func TestRevocationRegistry(t *testing.T) {
	reg := NewRevocationRegistry()

	nhi := &NHI{
		AgentID:   "red-agent-02",
		AgentType: AgentTypeRed,
		DID:       NewDID(),
	}

	if reg.IsRevoked(nhi.AgentID) {
		t.Fatal("agent should not be revoked initially")
	}

	rec := reg.Revoke(nhi, ReasonCompromised, "detected rogue behavior", "abm")
	if rec == nil {
		t.Fatal("Revoke returned nil")
	}

	if !reg.IsRevoked(nhi.AgentID) {
		t.Fatal("agent should be revoked after Revoke")
	}

	if !nhi.IsRevoked() {
		t.Fatal("NHI should be marked revoked")
	}

	if reg.Count() != 1 {
		t.Errorf("Count = %d, want 1", reg.Count())
	}

	// Verificar claims
	claims := &ClaimsNexus{
		AgentID: nhi.AgentID,
		DID:     nhi.DID.String(),
	}
	if err := reg.VerifyNotRevoked(claims); err == nil {
		t.Fatal("VerifyNotRevoked should fail for revoked agent")
	}
}

// TestAuditEntry firma y verifica una entrada de audit.
func TestAuditEntry(t *testing.T) {
	kp, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	nhi := &NHI{
		AgentID:   "red-agent-03",
		AgentType: AgentTypeRed,
		DID:       NewDID(),
	}

	entry := NewAuditEntry(nhi, ActionScan, "target.local", []byte("scan payload"), "ok", "")
	if entry.ID == "" {
		t.Fatal("entry ID is empty")
	}

	// Firmar
	if err := entry.Sign(kp, "test-key-1"); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if entry.Signature == "" {
		t.Fatal("signature is empty after Sign")
	}

	// Verificar
	ok, err := entry.Verify(kp)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !ok {
		t.Fatal("Verify returned false (SECURITY BUG)")
	}

	// Tamper y verificar que falla
	entry.Result = "denied" // modificar después de firmar
	ok, err = entry.Verify(kp)
	if err != nil {
		t.Fatalf("Verify after tamper failed: %v", err)
	}
	if ok {
		t.Fatal("Verify succeeded after tamper (SECURITY BUG)")
	}
}

// TestX509CertAuthority verifica emisión de certificados.
func TestX509CertAuthority(t *testing.T) {
	ca, err := NewCertAuthority()
	if err != nil {
		t.Fatalf("NewCertAuthority failed: %v", err)
	}

	did := NewDID()
	certPEM, err := ca.IssueCert("red-agent-04", AgentTypeRed, did, time.Hour)
	if err != nil {
		t.Fatalf("IssueCert failed: %v", err)
	}
	if len(certPEM) == 0 {
		t.Fatal("certificate PEM is empty")
	}

	// Verificar que está firmado por la CA
	if err := ca.VerifyCert(certPEM); err != nil {
		t.Fatalf("VerifyCert failed: %v", err)
	}

	// Parsear y verificar CN
	cert, err := ParseCert(certPEM)
	if err != nil {
		t.Fatalf("ParseCert failed: %v", err)
	}
	if cert.Subject.CommonName != "red-agent-04" {
		t.Errorf("CN = %q, want red-agent-04", cert.Subject.CommonName)
	}
}

// TestNHIPermissions verifica el sistema de permisos.
func TestNHIPermissions(t *testing.T) {
	nhi := &NHI{
		AllowedActions: []ActionType{ActionScan, ActionExploit},
	}

	if !nhi.HasPermission(ActionScan) {
		t.Error("expected SCAN permission")
	}
	if !nhi.HasPermission(ActionExploit) {
		t.Error("expected EXPLOIT permission")
	}
	if nhi.HasPermission(ActionDumpCredentials) {
		t.Error("did not expect DUMP_CREDENTIALS permission")
	}
}

// TestNHILifecycle verifica el ciclo de vida de NHI.
func TestNHILifecycle(t *testing.T) {
	now := time.Now()
	nhi := &NHI{
		AgentID:   "test-agent",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	if nhi.IsExpired() {
		t.Error("NHI should not be expired")
	}
	if nhi.IsRevoked() {
		t.Error("NHI should not be revoked")
	}
	if !nhi.IsValid() {
		t.Error("NHI should be valid")
	}

	// Expirar
	nhi.ExpiresAt = now.Add(-time.Hour)
	if !nhi.IsExpired() {
		t.Error("NHI should be expired")
	}
	if nhi.IsValid() {
		t.Error("NHI should not be valid (expired)")
	}

	// Revocar
	nhi.ExpiresAt = now.Add(time.Hour)
	nhi.Revoke("test reason")
	if !nhi.IsRevoked() {
		t.Error("NHI should be revoked")
	}
	if nhi.IsValid() {
		t.Error("NHI should not be valid (revoked)")
	}
}
