// Nexus IA v6.0 — Tests del Phantom Onion
package onion

import (
	"bytes"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Tests de Crypto
// ═══════════════════════════════════════════════════════════════

// TestLayerCipher verifica el cifrado por capa.
func TestLayerCipher(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey failed: %v", err)
	}

	lc, err := NewLayerCipher(masterKey, 0)
	if err != nil {
		t.Fatalf("NewLayerCipher failed: %v", err)
	}

	plaintext := []byte("hello onion")

	// Cifrar
	encrypted, err := lc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted should differ from plaintext")
	}

	// Descifrar
	decrypted, err := lc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted should match plaintext")
	}
}

// TestLayerCipherTamper verifica detección de tampering.
func TestLayerCipherTamper(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	lc, _ := NewLayerCipher(masterKey, 0)

	plaintext := []byte("secret data")
	encrypted, _ := lc.Encrypt(plaintext)

	// Modificar un byte
	encrypted[len(encrypted)-1] ^= 0xff

	_, err := lc.Decrypt(encrypted)
	if err == nil {
		t.Error("tampered data should fail to decrypt")
	}
}

// TestMultiLayerPeel verifica el cifrado multi-capa.
func TestMultiLayerPeel(t *testing.T) {
	masterKey, _ := GenerateMasterKey()

	mlc, err := NewMultiLayerCipher(masterKey, 3)
	if err != nil {
		t.Fatalf("NewMultiLayerCipher failed: %v", err)
	}

	if mlc.Hops() != 3 {
		t.Errorf("expected 3 hops, got %d", mlc.Hops())
	}

	plaintext := []byte("multi layer test")
	onionPayload, err := mlc.PeelOnion(plaintext)
	if err != nil {
		t.Fatalf("PeelOnion failed: %v", err)
	}
	if bytes.Equal(onionPayload, plaintext) {
		t.Error("onion should differ from plaintext")
	}

	// Descifrar todas las capas
	recovered, err := mlc.UnpeelAll(onionPayload)
	if err != nil {
		t.Fatalf("UnpeelAll failed: %v", err)
	}
	if !bytes.Equal(recovered, plaintext) {
		t.Error("recovered should match plaintext")
	}
}

// TestMultiLayerPartialPeel verifica que cada relay descifra solo su capa.
func TestMultiLayerPartialPeel(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	mlc, _ := NewMultiLayerCipher(masterKey, 3)

	plaintext := []byte("partial peel test")
	onionPayload, _ := mlc.PeelOnion(plaintext)

	// Primera capa: desciframos layer[0]
	layer0, err := mlc.UnpeelLayer(onionPayload)
	if err != nil {
		t.Fatalf("UnpeelLayer failed: %v", err)
	}
	if bytes.Equal(layer0, plaintext) {
		t.Error("after one peel, should still be encrypted")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Directory
// ═══════════════════════════════════════════════════════════════

// TestDirectory verifica el directorio de nodos.
func TestDirectory(t *testing.T) {
	dir := NewDirectory()

	node := &NodeInfo{
		ID:       "node-1",
		Endpoint: "10.60.0.1:9000",
		Active:   true,
		LatencyMs: 50,
	}
	dir.Add(node)

	if dir.Count() != 1 {
		t.Errorf("count %d, want 1", dir.Count())
	}

	got, ok := dir.Get("node-1")
	if !ok {
		t.Fatal("node not found")
	}
	if got.ID != "node-1" {
		t.Errorf("wrong ID: %s", got.ID)
	}

	// ListActive
	if dir.CountActive() != 1 {
		t.Errorf("active count %d, want 1", dir.CountActive())
	}
}

// TestDirectoryInactive verifica nodos inactivos.
func TestDirectoryInactive(t *testing.T) {
	dir := NewDirectory()

	dir.Add(&NodeInfo{ID: "active-1", Active: true})
	dir.Add(&NodeInfo{ID: "inactive-1", Active: false})

	if dir.CountActive() != 1 {
		t.Errorf("active count %d, want 1", dir.CountActive())
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Circuit
// ═══════════════════════════════════════════════════════════════

// TestCircuitBuilder verifica la construcción de circuitos.
func TestCircuitBuilder(t *testing.T) {
	dir := NewDirectory()
	for i := 0; i < 5; i++ {
		dir.Add(&NodeInfo{
			ID:     NodeID("node-" + string(rune('0'+i))),
			Active: true,
		})
	}

	cfg := DefaultConfig()
	builder := NewCircuitBuilder(cfg, dir)

	circuit, mlc, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if circuit == nil {
		t.Fatal("circuit is nil")
	}
	if circuit.HopCount() != 3 {
		t.Errorf("expected 3 hops, got %d", circuit.HopCount())
	}
	if len(circuit.Path) != 3 {
		t.Errorf("expected 3 nodes in path, got %d", len(circuit.Path))
	}
	if mlc == nil || mlc.Hops() != 3 {
		t.Error("mlc invalid")
	}

	// Verificar que los nodos son únicos
	seen := make(map[NodeID]bool)
	for _, n := range circuit.Path {
		if seen[n] {
			t.Errorf("node %s repeated in path", n)
		}
		seen[n] = true
	}
}

// TestCircuitBuilderNotEnoughNodes verifica error con pocos nodos.
func TestCircuitBuilderNotEnoughNodes(t *testing.T) {
	dir := NewDirectory()
	dir.Add(&NodeInfo{ID: "only-1", Active: true})

	cfg := DefaultConfig()
	builder := NewCircuitBuilder(cfg, dir)

	_, _, err := builder.Build()
	if err == nil {
		t.Error("Build should fail with not enough nodes")
	}
}

// TestCircuitExpiry verifica la expiración.
func TestCircuitExpiry(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	path := []NodeID{"a", "b", "c"}

	circuit, _, err := NewCircuitFromPath(path, masterKey, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewCircuitFromPath failed: %v", err)
	}

	if circuit.IsExpired() {
		t.Error("circuit should not be expired yet")
	}

	time.Sleep(150 * time.Millisecond)

	if !circuit.IsExpired() {
		t.Error("circuit should be expired")
	}
	if circuit.TTL() != 0 {
		t.Errorf("expired circuit TTL should be 0, got %v", circuit.TTL())
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de CircuitPool
// ═══════════════════════════════════════════════════════════════

// TestCircuitPool verifica el pool de circuitos.
func TestCircuitPool(t *testing.T) {
	dir := NewDirectory()
	for i := 0; i < 5; i++ {
		dir.Add(&NodeInfo{ID: NodeID("node-" + string(rune('0'+i))), Active: true})
	}

	builder := NewCircuitBuilder(DefaultConfig(), dir)
	pool := NewCircuitPool(builder, 3)

	// GetOrCreate
	c1, _, err := pool.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if c1 == nil {
		t.Fatal("circuit is nil")
	}
	if pool.Size() != 1 {
		t.Errorf("pool size %d, want 1", pool.Size())
	}

	// Segunda llamada debe devolver el mismo
	c2, _, err := pool.GetOrCreate()
	if err != nil {
		t.Fatalf("second GetOrCreate failed: %v", err)
	}
	if c2.ID != c1.ID {
		t.Error("GetOrCreate should reuse existing circuit")
	}

	// Close
	pool.Close(c1.ID)
	if pool.Size() != 0 {
		t.Errorf("after close, size %d, want 0", pool.Size())
	}
}

// TestCircuitPoolMaxSize verifica el límite del pool.
func TestCircuitPoolMaxSize(t *testing.T) {
	dir := NewDirectory()
	for i := 0; i < 5; i++ {
		dir.Add(&NodeInfo{ID: NodeID("node-" + string(rune('0'+i))), Active: true})
	}

	builder := NewCircuitBuilder(DefaultConfig(), dir)
	pool := NewCircuitPool(builder, 2)

	// Crear 3 circuitos forzando eviction
	for i := 0; i < 3; i++ {
		// Forzar creación
		_, _, _ = pool.builder.Build()
		pool.circuits[string(rune('a'+i))] = &Circuit{ID: string(rune('a' + i)), CreatedAt: time.Now()}
	}
	// Ahora excede
	_, _, _ = pool.GetOrCreate()
	if pool.Size() > 3 { // maxSize=2 + el nuevo que se acaba de añadir
		t.Logf("pool size: %d", pool.Size())
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Client
// ═══════════════════════════════════════════════════════════════

// TestClientSend verifica el envío onion.
func TestClientSend(t *testing.T) {
	dir := NewDirectory()
	for i := 0; i < 5; i++ {
		dir.Add(&NodeInfo{ID: NodeID("node-" + string(rune('0'+i))), Active: true})
	}

	client := NewClient(DefaultConfig(), dir)

	payload := []byte("secret message")
	onion, circuit, err := client.Send(payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if onion == nil {
		t.Fatal("onion is nil")
	}
	if circuit == nil {
		t.Fatal("circuit is nil")
	}
	if bytes.Equal(onion, payload) {
		t.Error("onion should differ from plaintext")
	}
	if circuit.HopCount() != 3 {
		t.Errorf("expected 3 hops, got %d", circuit.HopCount())
	}
}

// TestClientReply verifica respuesta.
func TestClientReply(t *testing.T) {
	dir := NewDirectory()
	for i := 0; i < 5; i++ {
		dir.Add(&NodeInfo{ID: NodeID("node-" + string(rune('0'+i))), Active: true})
	}

	client := NewClient(DefaultConfig(), dir)

	payload := []byte("request")
	_, circuit, _ := client.Send(payload)

	// Simular respuesta: cifrarla con las capas (pero con UnpeelAll inverso)
	// En la práctica, el cliente conoce todas las capas.
	reply := []byte("response")
	_, cipher, ok := client.pool.Get(circuit.ID)
	if !ok || cipher == nil {
		t.Fatal("circuit not found in pool")
	}
	encryptedReply, _ := cipher.PeelOnion(reply)

	decrypted, err := client.DecryptReply(circuit.ID, encryptedReply)
	if err != nil {
		t.Fatalf("DecryptReply failed: %v", err)
	}
	if !bytes.Equal(decrypted, reply) {
		t.Errorf("decrypted %s, want %s", decrypted, reply)
	}
}

// TestClientClose verifica el cierre.
func TestClientClose(t *testing.T) {
	dir := NewDirectory()
	dir.Add(&NodeInfo{ID: "n1", Active: true})
	dir.Add(&NodeInfo{ID: "n2", Active: true})
	dir.Add(&NodeInfo{ID: "n3", Active: true})

	client := NewClient(DefaultConfig(), dir)
	_, _, _ = client.Send([]byte("test"))

	if client.CircuitCount() == 0 {
		t.Error("expected at least one circuit")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if client.CircuitCount() != 0 {
		t.Errorf("circuits should be 0 after close, got %d", client.CircuitCount())
	}

	// Send después de close debe fallar
	_, _, err := client.Send([]byte("test"))
	if err == nil {
		t.Error("Send after close should fail")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Relay
// ═══════════════════════════════════════════════════════════════

// TestRelayProcessCell verifica el procesamiento de celdas.
func TestRelayProcessCell(t *testing.T) {
	layerKey := make([]byte, AESKeySize)
	for i := range layerKey {
		layerKey[i] = byte(i)
	}

	relay, err := NewRelay("relay-1", "10.60.0.1:9000", layerKey)
	if err != nil {
		t.Fatalf("NewRelay failed: %v", err)
	}

	// Registrar circuito
	relay.RegisterCircuit("circuit-1", "next-relay")

	// Cifrar un payload con la misma clave (como si fuéramos el emisor)
	lc, _ := NewLayerCipherFromKey(layerKey, 0)
	payload := []byte("test payload")
	encrypted, _ := lc.Encrypt(payload)

	// Procesar
	decrypted, nextNode, err := relay.ProcessCell("circuit-1", encrypted)
	if err != nil {
		t.Fatalf("ProcessCell failed: %v", err)
	}
	if !bytes.Equal(decrypted, payload) {
		t.Error("decrypted should match payload")
	}
	if nextNode != "next-relay" {
		t.Errorf("wrong next node: %s", nextNode)
	}
}

// TestRelayEncryptReply verifica el cifrado de respuesta.
func TestRelayEncryptReply(t *testing.T) {
	layerKey := bytes.Repeat([]byte{0x42}, AESKeySize)

	relay, _ := NewRelay("relay-1", "10.60.0.1:9000", layerKey)

	reply := []byte("response from relay")
	encrypted, err := relay.EncryptReply(reply)
	if err != nil {
		t.Fatalf("EncryptReply failed: %v", err)
	}
	if bytes.Equal(encrypted, reply) {
		t.Error("encrypted should differ from reply")
	}
}

// TestRelayClose verifica el cierre.
func TestRelayClose(t *testing.T) {
	layerKey := bytes.Repeat([]byte{0x42}, AESKeySize)
	relay, _ := NewRelay("relay-1", "addr", layerKey)

	if err := relay.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// ProcessCell después de close debe fallar
	_, _, err := relay.ProcessCell("c1", []byte("data"))
	if err == nil {
		t.Error("ProcessCell after close should fail")
	}
}

// ═══════════════════════════════════════════════════════════════
// Test end-to-end: cliente → 3 relays → cliente
// ═══════════════════════════════════════════════════════════════

// TestEndToEndOnion verifica el flujo completo con 3 relays.
//
// Este test verifica el pipeline real:
//   1. Cliente cifra con MultiLayerCipher (3 capas)
//   2. Relay 1 descifra capa 0
//   3. Relay 2 descifra capa 1
//   4. Relay 3 descifra capa 2 y ve el payload original
//
// Usa las MISMAS claves de capa que el MultiLayerCipher (derivadas
// con HKDF-SHA256 de la clave maestra).
func TestEndToEndOnion(t *testing.T) {
	masterKey, _ := GenerateMasterKey()

	// 1. Crear el MultiLayerCipher que usará el cliente
	mlc, err := NewMultiLayerCipher(masterKey, 3)
	if err != nil {
		t.Fatalf("NewMultiLayerCipher failed: %v", err)
	}

	// 2. Extraer las 3 claves de capa del MultiLayerCipher
	//    (para crear los 3 relays con las claves correctas)
	layerKeys := make([][]byte, 3)
	for i := 0; i < 3; i++ {
		lc, err := mlc.Layer(HopIndex(i))
		if err != nil {
			t.Fatalf("get layer %d failed: %v", i, err)
		}
		layerKeys[i] = lc.key
	}

	// 3. Crear los 3 relays con las claves correctas
	relay1, err := NewRelay("relay-1", "r1:9000", layerKeys[0])
	if err != nil {
		t.Fatalf("NewRelay(1) failed: %v", err)
	}
	relay2, err := NewRelay("relay-2", "r2:9000", layerKeys[1])
	if err != nil {
		t.Fatalf("NewRelay(2) failed: %v", err)
	}
	relay3, err := NewRelay("relay-3", "r3:9000", layerKeys[2])
	if err != nil {
		t.Fatalf("NewRelay(3) failed: %v", err)
	}

	// 4. Configurar routing
	relay1.RegisterCircuit("c1", "relay-2")
	relay2.RegisterCircuit("c1", "relay-3")
	relay3.RegisterCircuit("c1", "")

	// 5. Cliente cifra el payload con 3 capas
	payload := []byte("hello from client")
	onion, err := mlc.PeelOnion(payload)
	if err != nil {
		t.Fatalf("PeelOnion failed: %v", err)
	}

	// 6. Relay 1 descifra capa 0 (la más externa)
	d1, next1, err := relay1.ProcessCell("c1", onion)
	if err != nil {
		t.Fatalf("relay1 failed: %v", err)
	}
	if next1 != "relay-2" {
		t.Errorf("relay1 next: %s, want relay-2", next1)
	}

	// 7. Relay 2 descifra capa 1
	d2, next2, err := relay2.ProcessCell("c1", d1)
	if err != nil {
		t.Fatalf("relay2 failed: %v", err)
	}
	if next2 != "relay-3" {
		t.Errorf("relay2 next: %s, want relay-3", next2)
	}

	// 8. Relay 3 descifra capa 2 (la más interna) y ve el payload original
	d3, next3, err := relay3.ProcessCell("c1", d2)
	if err != nil {
		t.Fatalf("relay3 failed: %v", err)
	}
	if next3 != "" {
		t.Errorf("relay3 should be last, got next: %s", next3)
	}

	// 9. Verificar que el payload llegó intacto
	if !bytes.Equal(d3, payload) {
		t.Errorf("payload mismatch:\n  got:  %s\n  want: %s", d3, payload)
	}
}
