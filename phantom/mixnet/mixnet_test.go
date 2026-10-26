// Nexus IA v6.0 — Tests del Phantom Mixnet
package mixnet

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Tests de Crypto
// ═══════════════════════════════════════════════════════════════

// TestX25519 verifica X25519.
func TestX25519(t *testing.T) {
	alice, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("alice keypair: %v", err)
	}
	bob, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("bob keypair: %v", err)
	}

	aliceSS, _ := alice.SharedSecret(bob.PublicKey)
	bobSS, _ := bob.SharedSecret(alice.PublicKey)

	if !bytes.Equal(aliceSS, bobSS) {
		t.Error("shared secrets should match")
	}
}

// TestKyber768 verifica Kyber768.
func TestKyber768(t *testing.T) {
	kp, err := GenerateKyber768KeyPair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	ct, ss1, err := kp.Encapsulate()
	if err != nil {
		t.Fatalf("encapsulate: %v", err)
	}

	ss2, err := kp.Decapsulate(ct)
	if err != nil {
		t.Fatalf("decapsulate: %v", err)
	}

	if !bytes.Equal(ss1, ss2) {
		t.Error("shared secrets should match")
	}
}

// TestHybridKDF verifica el HKDF híbrido.
func TestHybridKDF(t *testing.T) {
	x25519SS := bytes.Repeat([]byte{1}, 32)
	kyberSS := bytes.Repeat([]byte{2}, 32)

	key1, err := HybridKDF(x25519SS, kyberSS, "test")
	if err != nil {
		t.Fatalf("HybridKDF: %v", err)
	}
	if len(key1) != AESKeySize {
		t.Errorf("wrong key size: %d", len(key1))
	}

	key2, _ := HybridKDF(x25519SS, kyberSS, "test")
	if !bytes.Equal(key1, key2) {
		t.Error("deterministic KDF failed")
	}
}

// TestLayerCipher verifica AES-256-GCM.
func TestLayerCipher(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, AESKeySize)
	lc, err := NewLayerCipher(key, 0)
	if err != nil {
		t.Fatalf("NewLayerCipher: %v", err)
	}

	plaintext := []byte("hello mixnet")
	encrypted, err := lc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := lc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted should match")
	}

	// Tamper
	encrypted[len(encrypted)-1] ^= 0xff
	_, err = lc.Decrypt(encrypted)
	if err == nil {
		t.Error("tampered data should fail")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Directory
// ═══════════════════════════════════════════════════════════════

// TestMixDirectory verifica el directorio.
func TestMixDirectory(t *testing.T) {
	dir := NewMixDirectory()
	dir.Add(&MixNode{ID: "mix-1", Active: true})
	dir.Add(&MixNode{ID: "mix-2", Active: true})
	dir.Add(&MixNode{ID: "mix-3", Active: false})

	if dir.Count() != 3 {
		t.Errorf("count %d, want 3", dir.Count())
	}
	if dir.CountActive() != 2 {
		t.Errorf("active count %d, want 2", dir.CountActive())
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Circuit
// ═══════════════════════════════════════════════════════════════

// TestMixCircuitBuilder verifica la construcción.
func TestMixCircuitBuilder(t *testing.T) {
	dir := NewMixDirectory()
	for i := 0; i < 8; i++ {
		dir.Add(&MixNode{ID: NodeID("mix-" + string(rune('0'+i))), Active: true})
	}

	cfg := DefaultConfig()
	builder := NewMixCircuitBuilder(cfg, dir)

	circuit, layerKeys, err := builder.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if circuit.HopCount() != 5 {
		t.Errorf("expected 5 hops, got %d", circuit.HopCount())
	}
	if len(layerKeys) != 5 {
		t.Errorf("expected 5 layer keys, got %d", len(layerKeys))
	}

	// Nodos únicos
	seen := make(map[NodeID]bool)
	for _, n := range circuit.Path {
		if seen[n] {
			t.Errorf("node %s repeated", n)
		}
		seen[n] = true
	}
}

// TestMixCircuitNotEnoughNodes verifica error.
func TestMixCircuitNotEnoughNodes(t *testing.T) {
	dir := NewMixDirectory()
	dir.Add(&MixNode{ID: "only-1", Active: true})

	builder := NewMixCircuitBuilder(DefaultConfig(), dir)
	_, _, err := builder.Build()
	if err == nil {
		t.Error("should fail with not enough nodes")
	}
}

// TestMixCircuitPool verifica el pool.
func TestMixCircuitPool(t *testing.T) {
	dir := NewMixDirectory()
	for i := 0; i < 8; i++ {
		dir.Add(&MixNode{ID: NodeID("mix-" + string(rune('0'+i))), Active: true})
	}

	builder := NewMixCircuitBuilder(DefaultConfig(), dir)
	pool := NewMixCircuitPool(builder, 5)

	c, err := pool.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if c == nil {
		t.Fatal("circuit is nil")
	}
	if pool.Size() != 1 {
		t.Errorf("size %d, want 1", pool.Size())
	}

	// Segunda llamada reutiliza
	c2, _ := pool.GetOrCreate()
	if c2.ID != c.ID {
		t.Error("should reuse circuit")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Sphinx
// ═══════════════════════════════════════════════════════════════

// TestSphinxBuild verifica la construcción de un Sphinx packet.
func TestSphinxBuild(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	sb := NewSphinxBuilder(masterKey, 5)
	packet, err := sb.BuildPacket([]byte("hello sphinx"))
	if err != nil {
		t.Fatalf("BuildPacket: %v", err)
	}

	if packet == nil {
		t.Fatal("packet is nil")
	}
	if len(packet.EphemeralPublicKey) != 32 {
		t.Errorf("ephemeral key size: %d", len(packet.EphemeralPublicKey))
	}
	// Header completo debe ser HeaderSize
	headerSize := len(packet.EphemeralPublicKey) + len(packet.Kyber768Ciphertext) +
		len(packet.RoutingInfo) + len(packet.MACs)
	if headerSize != HeaderSize {
		t.Errorf("header size: %d, want %d", headerSize, HeaderSize)
	}
	if len(packet.Payload) != PayloadSize {
		t.Errorf("payload size: %d, want %d", len(packet.Payload), PayloadSize)
	}
	if packet.Size() != PacketSize {
		t.Errorf("packet size: %d, want %d", packet.Size(), PacketSize)
	}
	if !packet.IsValidSize() {
		t.Error("IsValidSize should return true")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Mixing (el corazón del mixnet)
// ═══════════════════════════════════════════════════════════════

// TestMixerAdd verifica añadir paquetes.
func TestMixerAdd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnablePoissonDelays = false
	mixer := NewMixer(cfg)

	packet := &SphinxPacket{
		EphemeralPublicKey: make([]byte, 32),
		Payload:            make([]byte, 100),
	}

	if err := mixer.Add(packet); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if mixer.PendingCount() != 1 {
		t.Errorf("pending %d, want 1", mixer.PendingCount())
	}
}

// TestMixerBatch verifica el batching.
func TestMixerBatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 5
	cfg.EnablePoissonDelays = false

	mixer := NewMixer(cfg)

	received := make(chan []*SphinxPacket, 1)
	mixer.SetBatchReadyCallback(func(batch []*SphinxPacket) {
		select {
		case received <- batch:
		default:
		}
	})

	// Añadir exactamente batchSize
	for i := 0; i < 5; i++ {
		_ = mixer.Add(&SphinxPacket{Payload: make([]byte, 100)})
	}

	select {
	case batch := <-received:
		if len(batch) != 5 {
			t.Errorf("batch size %d, want 5", len(batch))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("batch should be ready")
	}

	stats := mixer.Stats()
	if stats.TotalMixed != 5 {
		t.Errorf("mixed %d, want 5", stats.TotalMixed)
	}
	if stats.TotalBatches != 1 {
		t.Errorf("batches %d, want 1", stats.TotalBatches)
	}
}

// TestPoissonDelayDistribution verifica la distribución Poisson.
func TestPoissonDelayDistribution(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnablePoissonDelays = true
	mixer := NewMixer(cfg)

	// Generar 1000 delays y verificar que caen en el rango
	delays := make([]time.Duration, 1000)
	for i := 0; i < 1000; i++ {
		delays[i] = mixer.poissonDelay()
	}

	// Verificar que todos están en rango
	for _, d := range delays {
		if d < MinDelay {
			t.Errorf("delay %v < MinDelay %v", d, MinDelay)
		}
		if d > MaxDelay {
			t.Errorf("delay %v > MaxDelay %v", d, MaxDelay)
		}
	}

	// Verificar que hay variabilidad (no son todos iguales)
	unique := make(map[time.Duration]int)
	for _, d := range delays {
		unique[d]++
	}
	if len(unique) < 10 {
		t.Errorf("expected variability, got %d unique delays", len(unique))
	}
}

// TestMixerShuffling verifica la reordenación.
func TestMixerShuffling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 10
	cfg.EnablePoissonDelays = false

	mixer := NewMixer(cfg)

	var receivedBatch []*SphinxPacket
	done := make(chan struct{})
	mixer.SetBatchReadyCallback(func(batch []*SphinxPacket) {
		receivedBatch = batch
		close(done)
	})

	// Añadir 10 paquetes con IDs distintos
	for i := 0; i < 10; i++ {
		p := &SphinxPacket{
			Payload: make([]byte, 100),
		}
		p.Payload[0] = byte(i) // marcar con ID
		_ = mixer.Add(p)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("batch timeout")
	}

	// Verificar que se recibieron 10
	if len(receivedBatch) != 10 {
		t.Errorf("batch size %d, want 10", len(receivedBatch))
	}
}

// TestCoverTraffic verifica el generador de cover traffic.
func TestCoverTraffic(t *testing.T) {
	gen := NewCoverTrafficGenerator(10.0) // 10 pkt/s

	var total atomic.Int64
	gen.SetPacketCallback(func(p *SphinxPacket) {
		total.Add(1)
	})

	gen.Start()
	time.Sleep(500 * time.Millisecond)
	_ = gen.Close()

	// Debería haber generado ~5 paquetes en 500ms
	if n := total.Load(); n < 2 {
		t.Errorf("expected at least 2 packets, got %d", n)
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de Client
// ═══════════════════════════════════════════════════════════════

// TestClientSend verifica el envío.
func TestClientSend(t *testing.T) {
	dir := NewMixDirectory()
	for i := 0; i < 8; i++ {
		dir.Add(&MixNode{ID: NodeID("mix-" + string(rune('0'+i))), Active: true})
	}

	cfg := DefaultConfig()
	cfg.EnableCoverTraffic = false
	cfg.EnablePoissonDelays = false

	client := NewClient(cfg, dir)

	packet, circuit, err := client.Send([]byte("hello mixnet"))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if packet == nil {
		t.Fatal("packet is nil")
	}
	if circuit == nil {
		t.Fatal("circuit is nil")
	}
	if circuit.HopCount() != 5 {
		t.Errorf("expected 5 hops, got %d", circuit.HopCount())
	}

	stats := client.Stats()
	if stats.TotalPacketsSent == 0 {
		t.Error("packets sent should be > 0")
	}
}

// ═══════════════════════════════════════════════════════════════
// Tests de MixNode
// ═══════════════════════════════════════════════════════════════

// TestMixNodeProcess verifica el procesamiento.
func TestMixNodeProcess(t *testing.T) {
	layerKey := bytes.Repeat([]byte{0x42}, AESKeySize)
	cfg := DefaultConfig()
	cfg.EnablePoissonDelays = false

	node, err := NewMixNodeRunner("mix-1", "addr", layerKey, cfg)
	if err != nil {
		t.Fatalf("NewMixNodeRunner: %v", err)
	}
	node.Start()

	// Procesar un paquete
	packet := &SphinxPacket{
		EphemeralPublicKey: make([]byte, 32),
		RoutingInfo:        make([]byte, HeaderSize),
		Payload:            make([]byte, 100),
	}

	if err := node.ProcessPacket("circuit-1", packet); err != nil {
		t.Fatalf("ProcessPacket: %v", err)
	}

	stats := node.Stats()
	if stats.PacketsReceived != 1 {
		t.Errorf("received %d, want 1", stats.PacketsReceived)
	}

	_ = node.Close()
}

// TestMixNodeClose verifica el cierre.
func TestMixNodeClose(t *testing.T) {
	layerKey := bytes.Repeat([]byte{0x42}, AESKeySize)
	node, _ := NewMixNodeRunner("mix-1", "addr", layerKey, DefaultConfig())

	if err := node.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// ProcessPacket después de close debe fallar
	err := node.ProcessPacket("c1", &SphinxPacket{})
	if err == nil {
		t.Error("ProcessPacket after close should fail")
	}
}

// ═══════════════════════════════════════════════════════════════
// Test End-to-End
// ═══════════════════════════════════════════════════════════════

// TestEndToEndMixnet verifica el flujo completo.
func TestEndToEndMixnet(t *testing.T) {
	// Directorio con 8 nodos
	dir := NewMixDirectory()
	for i := 0; i < 8; i++ {
		dir.Add(&MixNode{ID: NodeID("mix-" + string(rune('0'+i))), Active: true})
	}

	cfg := DefaultConfig()
	cfg.EnableCoverTraffic = false
	cfg.EnablePoissonDelays = false

	client := NewClient(cfg, dir)

	// Enviar 20 mensajes
	for i := 0; i < 20; i++ {
		payload := []byte("message " + string(rune('0'+i%10)))
		_, _, err := client.Send(payload)
		if err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}

	stats := client.Stats()
	if stats.TotalPacketsSent < 20 {
		t.Errorf("sent %d, want >= 20", stats.TotalPacketsSent)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
