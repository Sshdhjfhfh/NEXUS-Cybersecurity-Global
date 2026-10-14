// Nexus IA v6.0 — Tests del Phantom VPN
package vpn

import (
	"bytes"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Tests de Crypto
// ═══════════════════════════════════════════════════════════════

// TestX25519KeyPair verifica la generación de claves X25519.
func TestX25519KeyPair(t *testing.T) {
	kp, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateX25519KeyPair failed: %v", err)
	}
	if isAllZero(kp.PublicKey[:]) {
		t.Error("public key is all zero")
	}
	if isAllZero(kp.PrivateKey[:]) {
		t.Error("private key is all zero")
	}
}

// TestX25519SharedSecret verifica el ECDH.
func TestX25519SharedSecret(t *testing.T) {
	alice, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("alice keypair failed: %v", err)
	}
	bob, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("bob keypair failed: %v", err)
	}

	aliceSS, err := alice.SharedSecret(bob.PublicKey)
	if err != nil {
		t.Fatalf("alice shared secret failed: %v", err)
	}
	bobSS, err := bob.SharedSecret(alice.PublicKey)
	if err != nil {
		t.Fatalf("bob shared secret failed: %v", err)
	}

	if !bytes.Equal(aliceSS, bobSS) {
		t.Error("shared secrets should match")
	}
}

// TestKyber768KeyPair verifica Kyber768.
func TestKyber768KeyPair(t *testing.T) {
	kp, err := GenerateKyber768KeyPair()
	if err != nil {
		t.Fatalf("GenerateKyber768KeyPair failed: %v", err)
	}
	if kp.PublicKey == nil || kp.PrivateKey == nil {
		t.Fatal("keypair has nil components")
	}
}

// TestKyber768Encapsulate verifica el KEM.
func TestKyber768Encapsulate(t *testing.T) {
	kp, err := GenerateKyber768KeyPair()
	if err != nil {
		t.Fatalf("keypair failed: %v", err)
	}

	ct, ss1, err := kp.Encapsulate()
	if err != nil {
		t.Fatalf("encapsulate failed: %v", err)
	}
	if len(ct) == 0 || len(ss1) == 0 {
		t.Fatal("ciphertext or shared secret is empty")
	}

	ss2, err := kp.Decapsulate(ct)
	if err != nil {
		t.Fatalf("decapsulate failed: %v", err)
	}

	if !bytes.Equal(ss1, ss2) {
		t.Error("shared secrets should match after decapsulation")
	}
}

// TestHybridSharedSecret verifica el HKDF híbrido.
func TestHybridSharedSecret(t *testing.T) {
	x25519Secret := bytes.Repeat([]byte{1}, 32)
	kyberSecret := bytes.Repeat([]byte{2}, 32)

	key1, err := HybridSharedSecret(x25519Secret, kyberSecret, "test")
	if err != nil {
		t.Fatalf("HybridSharedSecret failed: %v", err)
	}
	if len(key1) != SessionKeySize {
		t.Errorf("key size %d, expected %d", len(key1), SessionKeySize)
	}

	// Mismo input → misma key (determinista)
	key2, err := HybridSharedSecret(x25519Secret, kyberSecret, "test")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if !bytes.Equal(key1, key2) {
		t.Error("same inputs should produce same key")
	}

	// Diferente info → diferente key
	key3, err := HybridSharedSecret(x25519Secret, kyberSecret, "different")
	if err != nil {
		t.Fatalf("third call failed: %v", err)
	}
	if bytes.Equal(key1, key3) {
		t.Error("different info should produce different key")
	}
}

// TestAEAD verifica ChaCha20-Poly1305.
func TestAEAD(t *testing.T) {
	key := bytes.Repeat([]byte{42}, ChaCha20Poly1305KeySize)
	aead, err := NewAEAD(key)
	if err != nil {
		t.Fatalf("NewAEAD failed: %v", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	plaintext := []byte("Hello, Phantom VPN!")
	additionalData := []byte("header")

	ciphertext, err := aead.Encrypt(nonce, plaintext, additionalData)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := aead.Decrypt(nonce, ciphertext, additionalData)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted should match original")
	}

	// Tamper
	ciphertext[0] ^= 0xff
	_, err = aead.Decrypt(nonce, ciphertext, additionalData)
	if err == nil {
		t.Error("tampered ciphertext should fail to decrypt")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Handshake
// ═══════════════════════════════════════════════════════════════

// TestHandshakeFull verifica el handshake completo.
func TestHandshakeFull(t *testing.T) {
	// 1. Initiation
	initiator, err := NewInitiatorHandshake()
	if err != nil {
		t.Fatalf("NewInitiatorHandshake failed: %v", err)
	}
	initMsg, err := initiator.CreateInitiation()
	if err != nil {
		t.Fatalf("CreateInitiation failed: %v", err)
	}

	// 2. Responder procesa
	responder, err := NewResponderHandshake()
	if err != nil {
		t.Fatalf("NewResponderHandshake failed: %v", err)
	}
	if err := responder.ProcessInitiation(initMsg); err != nil {
		t.Fatalf("ProcessInitiation failed: %v", err)
	}

	// 3. Response
	respMsg, err := responder.CreateResponse()
	if err != nil {
		t.Fatalf("CreateResponse failed: %v", err)
	}
	responderCiphertext, err := responder.FinalizeResponder()
	if err != nil {
		t.Fatalf("FinalizeResponder failed: %v", err)
	}

	// 4. Initiator procesa
	if err := initiator.ProcessResponse(respMsg); err != nil {
		t.Fatalf("ProcessResponse failed: %v", err)
	}
	if err := initiator.FinalizeInitiator(responderCiphertext); err != nil {
		t.Fatalf("FinalizeInitiator failed: %v", err)
	}

	// 5. Verificar que ambos tienen la misma session key
	if initiator.SessionKey == nil || responder.SessionKey == nil {
		t.Fatal("session key is nil")
	}
	if !bytes.Equal(initiator.SessionKey, responder.SessionKey) {
		t.Error("session keys should match")
	}
}

// TestHandshakeReplay verifica la protección anti-replay.
func TestHandshakeReplay(t *testing.T) {
	initiator, _ := NewInitiatorHandshake()
	initMsg, _ := initiator.CreateInitiation()

	responder, _ := NewResponderHandshake()
	if err := responder.ProcessInitiation(initMsg); err != nil {
		t.Fatalf("first initiation failed: %v", err)
	}

	// Replay: mismo nonce → debe fallar
	err := responder.ProcessInitiation(initMsg)
	if err == nil {
		t.Error("replay should be rejected")
	}
}

// TestHandshakeTimestampDrift verifica el anti-timestamp.
func TestHandshakeTimestampDrift(t *testing.T) {
	initiator, _ := NewInitiatorHandshake()
	initMsg, _ := initiator.CreateInitiation()

	// Modificar timestamp a muy antiguo
	initMsg.Timestamp = time.Now().Add(-1 * time.Hour).UnixNano()

	responder, _ := NewResponderHandshake()
	err := responder.ProcessInitiation(initMsg)
	if err == nil {
		t.Error("old timestamp should be rejected")
	}
}

// TestHandshakeInvalidType verifica validación de tipo.
func TestHandshakeInvalidType(t *testing.T) {
	initiator, _ := NewInitiatorHandshake()
	initMsg, _ := initiator.CreateInitiation()
	initMsg.Type = 99 // inválido

	responder, _ := NewResponderHandshake()
	err := responder.ProcessInitiation(initMsg)
	if err == nil {
		t.Error("invalid type should be rejected")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Pool
// ═══════════════════════════════════════════════════════════════

// TestPoolAdd verifica añadir nodos.
func TestPoolAdd(t *testing.T) {
	pool := NewPool()

	node := &Node{
		ID:       "node-1",
		Name:     "Test Node 1",
		Zone:     ZoneA,
		Endpoint: "10.60.0.1:51820",
		State:    NodeActive,
	}
	if err := pool.Add(node); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if pool.Count() != 1 {
		t.Errorf("count %d, expected 1", pool.Count())
	}
}

// TestPoolSelectBest verifica la selección por latencia.
func TestPoolSelectBest(t *testing.T) {
	pool := NewPool()

	pool.Add(&Node{ID: "slow", Zone: ZoneA, Endpoint: "a:1", State: NodeActive, LastLatencyMs: 200})
	pool.Add(&Node{ID: "fast", Zone: ZoneB, Endpoint: "b:1", State: NodeActive, LastLatencyMs: 50})
	pool.Add(&Node{ID: "medium", Zone: ZoneC, Endpoint: "c:1", State: NodeActive, LastLatencyMs: 100})

	best := pool.SelectBest()
	if best == nil {
		t.Fatal("SelectBest returned nil")
	}
	if best.ID != "fast" {
		t.Errorf("expected 'fast', got %s", best.ID)
	}
}

// TestPoolSkipsDown verifica que se saltan nodos caídos.
func TestPoolSkipsDown(t *testing.T) {
	pool := NewPool()

	pool.Add(&Node{ID: "down", Zone: ZoneA, Endpoint: "a:1", State: NodeDown, LastLatencyMs: 10})
	pool.Add(&Node{ID: "active", Zone: ZoneB, Endpoint: "b:1", State: NodeActive, LastLatencyMs: 100})

	best := pool.SelectBest()
	if best.ID != "active" {
		t.Errorf("expected 'active', got %s", best.ID)
	}
}

// TestPoolByZone verifica la selección por zona.
func TestPoolByZone(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "a1", Zone: ZoneA, Endpoint: "a:1", State: NodeActive, LastLatencyMs: 100})
	pool.Add(&Node{ID: "b1", Zone: ZoneB, Endpoint: "b:1", State: NodeActive, LastLatencyMs: 50})

	best := pool.SelectByZone(ZoneA)
	if best == nil {
		t.Fatal("SelectByZone returned nil")
	}
	if best.ID != "a1" {
		t.Errorf("expected 'a1', got %s", best.ID)
	}
}

// TestPoolStats verifica las estadísticas.
func TestPoolStats(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "a", Zone: ZoneA, Endpoint: "a:1", State: NodeActive, LastLatencyMs: 100})
	pool.Add(&Node{ID: "b", Zone: ZoneB, Endpoint: "b:1", State: NodeActive, LastLatencyMs: 200})
	pool.Add(&Node{ID: "c", Zone: ZoneA, Endpoint: "c:1", State: NodeDown})

	stats := pool.Stats()
	if stats.TotalNodes != 3 {
		t.Errorf("TotalNodes = %d, want 3", stats.TotalNodes)
	}
	if stats.ByZone[ZoneA] != 2 {
		t.Errorf("ZoneA count = %d, want 2", stats.ByZone[ZoneA])
	}
	if stats.AvgLatencyMs != 150 {
		t.Errorf("AvgLatencyMs = %d, want 150", stats.AvgLatencyMs)
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Client (orquestador)
// ═══════════════════════════════════════════════════════════════

// TestClientConnect verifica la conexión completa.
func TestClientConnect(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "n1", Zone: ZoneA, Endpoint: "n1:51820", State: NodeActive, LastLatencyMs: 50})

	client := NewClient(DefaultConfig(), pool)
	session, err := client.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if session == nil {
		t.Fatal("session is nil")
	}
	if session.NodeID != "n1" {
		t.Errorf("wrong node: %s", session.NodeID)
	}
	if len(session.SessionKey) != SessionKeySize {
		t.Errorf("wrong session key size: %d", len(session.SessionKey))
	}
	if session.IsClosed() {
		t.Error("new session should not be closed")
	}
}

// TestClientEncryptDecrypt verifica cifrado end-to-end.
func TestClientEncryptDecrypt(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "n1", Zone: ZoneA, Endpoint: "n1:51820", State: NodeActive, LastLatencyMs: 50})

	client := NewClient(DefaultConfig(), pool)
	session, err := client.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Cifrar
	plaintext := []byte("secret message")
	nonce, ciphertext, err := session.Encrypt(plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Descifrar
	decrypted, err := session.Decrypt(nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted should match original")
	}

	// Verificar contadores
	stats := session.Stats()
	if stats.BytesSent == 0 || stats.BytesReceived == 0 {
		t.Error("bytes sent/received should be > 0")
	}
}

// TestClientNoNodes verifica error sin nodos.
func TestClientNoNodes(t *testing.T) {
	pool := NewPool()
	client := NewClient(DefaultConfig(), pool)

	_, err := client.Connect()
	if err == nil {
		t.Error("Connect should fail without nodes")
	}
}

// TestClientCloseSession verifica cerrar una sesión.
func TestClientCloseSession(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "n1", Zone: ZoneA, Endpoint: "n1:51820", State: NodeActive})

	client := NewClient(DefaultConfig(), pool)
	session, _ := client.Connect()

	if err := client.CloseSession(session.ID); err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	// Verificar que la sesión está cerrada
	if !session.IsClosed() {
		t.Error("session should be closed")
	}

	// Verificar que ya no está en el mapa
	if client.SessionCount() != 0 {
		t.Errorf("session count %d, expected 0", client.SessionCount())
	}
}

// TestClientClose verifica cerrar el cliente completo.
func TestClientClose(t *testing.T) {
	pool := NewPool()
	pool.Add(&Node{ID: "n1", Zone: ZoneA, Endpoint: "n1:51820", State: NodeActive})

	client := NewClient(DefaultConfig(), pool)
	_, _ = client.Connect()
	_, _ = client.Connect()

	if client.SessionCount() != 2 {
		t.Errorf("expected 2 sessions, got %d", client.SessionCount())
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if client.SessionCount() != 0 {
		t.Errorf("all sessions should be closed")
	}
}

// TestClientMultipleNodes verifica con varios nodos.
func TestClientMultipleNodes(t *testing.T) {
	pool := NewPool()
	for i := 0; i < 5; i++ {
		pool.Add(&Node{
			ID:            "node-" + string(rune('0'+i)),
			Zone:          ZoneA,
			Endpoint:      "n:51820",
			State:         NodeActive,
			LastLatencyMs: int64(100 - i*10),
		})
	}

	client := NewClient(DefaultConfig(), pool)
	session, err := client.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Debe elegir el más rápido (node-4, latencia 60)
	if session.NodeID != "node-4" {
		t.Errorf("expected node-4 (fastest), got %s", session.NodeID)
	}
}
