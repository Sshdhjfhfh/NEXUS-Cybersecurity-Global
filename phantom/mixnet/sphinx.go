// Nexus IA v6.0 — Phantom Mixnet — Sphinx packet format
//
// Sphinx es el formato de paquete anónimo más avanzado. Propiedades:
//   - Tamaño FIJO (anti-fingerprinting por tamaño)
//   - Header cifrado en capas (cada hop descifra su parte)
//   - MACs encadenados (integridad por hop)
//   - Routing info enmascarada (cada hop solo ve el siguiente)
//
// Estructura de un Sphinx packet:
//
//   ┌─────────────────────────────────────────────┐
//   │ Ephemeral X25519 pubkey (32 bytes)          │
//   ├─────────────────────────────────────────────┤
//   │ Kyber768 ciphertext (variable)              │
//   ├─────────────────────────────────────────────┤
//   │ Routing info (cifrado en capas)             │
//   ├─────────────────────────────────────────────┤
//   │ MACs por hop (integridad)                   │
//   ├─────────────────────────────────────────────┤
//   │ Payload cifrado end-to-end                  │
//   └─────────────────────────────────────────────┘
package mixnet

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// Sphinx Builder
// ═══════════════════════════════════════════════════════════════

// SphinxBuilder construye Sphinx packets.
type SphinxBuilder struct {
	masterKey []byte
	hops      int
}

// NewSphinxBuilder crea un builder con una master key.
func NewSphinxBuilder(masterKey []byte, hops int) *SphinxBuilder {
	return &SphinxBuilder{
		masterKey: masterKey,
		hops:      hops,
	}
}

// BuildPacket construye un Sphinx packet para un payload.
//
//   1. Genera ephemeral X25519 keypair
//   2. Cifra routing info capa por capa (de fuera hacia dentro)
//   3. Cifra payload end-to-end
//   4. Construye MACs por hop
//   5. Rellena a PacketSize
func (sb *SphinxBuilder) BuildPacket(payload []byte) (*SphinxPacket, error) {
	// 1. Ephemeral X25519
	ephemeral, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("ephemeral keypair: %w", err)
	}

	// 2. Routing info (placeholder en esta implementación)
	routingInfo := sb.buildRoutingInfo()

	// 3. Cifrar routing info capa por capa (de dentro hacia fuera)
	encryptedRouting := routingInfo
	for i := sb.hops - 1; i >= 0; i-- {
		layerKey := sb.deriveLayerKey(i)
		lc, err := NewLayerCipher(layerKey, HopIndex(i))
		if err != nil {
			return nil, fmt.Errorf("layer cipher %d: %w", i, err)
		}
		encryptedRouting, err = lc.Encrypt(encryptedRouting)
		if err != nil {
			return nil, fmt.Errorf("encrypt routing layer %d: %w", i, err)
		}
	}

	// 4. Cifrar payload end-to-end con la última capa
	payloadKey := sb.deriveLayerKey(sb.hops - 1)
	payloadCipher, err := NewLayerCipher(payloadKey, HopIndex(sb.hops-1))
	if err != nil {
		return nil, fmt.Errorf("payload cipher: %w", err)
	}
	encryptedPayload, err := payloadCipher.Encrypt(payload)
	if err != nil {
		return nil, fmt.Errorf("encrypt payload: %w", err)
	}

	// 5. MACs por hop
	macs := sb.buildMACs()

	// 6. Padding a PayloadSize
	paddedPayload := make([]byte, PayloadSize)
	copy(paddedPayload, encryptedPayload)

	// 7. Construir packet con tamaños controlados
	// Header = EphemeralPublicKey(32) + Kyber768Ciphertext(1088) + RoutingInfo + MACs
	// Debe sumar exactamente HeaderSize (4096)
	const ephSize = 32
	const kyberCTSize = 1088
	routingSize := HeaderSize - ephSize - kyberCTSize - len(macs)
	if routingSize < 0 {
		routingSize = 0
	}

	packet := &SphinxPacket{
		EphemeralPublicKey: ephemeral.PublicKey[:ephSize],
		Kyber768Ciphertext: sb.deriveKyberCiphertext()[:kyberCTSize],
		RoutingInfo:        padTo(encryptedRouting, routingSize),
		MACs:               macs,
		Payload:            paddedPayload,
	}

	return packet, nil
}

// buildRoutingInfo construye la routing info original.
// En producción, contendría la lista de nodos siguientes.
func (sb *SphinxBuilder) buildRoutingInfo() []byte {
	info := make([]byte, 512)
	// En esta implementación, solo llenamos con datos dummy
	// La routing info real se añadiría aquí
	for i := range info {
		info[i] = byte(i % 256)
	}
	return info
}

// buildMACs construye los MACs por hop.
func (sb *SphinxBuilder) buildMACs() []byte {
	// En producción, cada hop tendría su MAC
	// Aquí generamos N MACs de 32 bytes cada uno
	macs := make([]byte, sb.hops*32)
	for i := 0; i < sb.hops; i++ {
		mac := sb.deriveMAC(i)
		copy(macs[i*32:(i+1)*32], mac)
	}
	return macs
}

// deriveLayerKey deriva la clave de capa i.
func (sb *SphinxBuilder) deriveLayerKey(hop int) []byte {
	info := []byte(fmt.Sprintf("mixnet-layer-%d", hop))
	hkdfReader := hkdf.New(sha256.New, sb.masterKey, nil, info)
	key := make([]byte, AESKeySize)
	_, _ = io.ReadFull(hkdfReader, key)
	return key
}

// deriveMAC deriva el MAC del hop i.
func (sb *SphinxBuilder) deriveMAC(hop int) []byte {
	info := []byte(fmt.Sprintf("mixnet-mac-%d", hop))
	hkdfReader := hkdf.New(sha256.New, sb.masterKey, nil, info)
	mac := make([]byte, 32)
	_, _ = io.ReadFull(hkdfReader, mac)
	return mac
}

// deriveKyberCiphertext deriva un "ciphertext" dummy de Kyber.
func (sb *SphinxBuilder) deriveKyberCiphertext() []byte {
	ct := make([]byte, 1088) // Kyber768 ciphertext size
	_, _ = io.ReadFull(cryptorand.Reader, ct)
	return ct
}

// ═══════════════════════════════════════════════════════════════
// Sphinx Parser
// ═══════════════════════════════════════════════════════════════

// SphinxParser extrae información de un Sphinx packet en cada hop.
type SphinxParser struct {
	hopKey []byte
}

// NewSphinxParser crea un parser con la clave de un hop.
func NewSphinxParser(hopKey []byte) *SphinxParser {
	return &SphinxParser{hopKey: hopKey}
}

// ProcessPacket procesa un Sphinx packet en un hop.
//
// Devuelve:
//   - payload (si es el último hop)
//   - siguiente hop ID (si no es el último)
//
// En esta implementación simplificada, solo descifra la capa
// más externa del routing info y devuelve el payload descifrado.
func (sp *SphinxParser) ProcessPacket(packet *SphinxPacket) ([]byte, NodeID, error) {
	if packet == nil {
		return nil, "", errors.New(ErrInvalidPacket)
	}

	// Crear LayerCipher con la clave del hop
	lc, err := NewLayerCipher(sp.hopKey, 0)
	if err != nil {
		return nil, "", fmt.Errorf("layer cipher: %w", err)
	}

	// Descifrar routing info (la capa más externa)
	_, err = lc.Decrypt(packet.RoutingInfo[:min(len(packet.RoutingInfo), GCMNonceSize+512+GCMTagSize)])
	if err != nil {
		// Si falla, intentar con el payload (último hop)
		payload, err2 := lc.Decrypt(packet.Payload[:min(len(packet.Payload), GCMNonceSize+len(packet.Payload)-GCMTagSize)])
		if err2 == nil {
			return payload, "", nil
		}
		return nil, "", fmt.Errorf("%s: %w", ErrDecryptionFailed, err)
	}

	// Si descifró routing info, reenviar al siguiente nodo
	nextHop := NodeID("next-hop") // Placeholder
	return nil, nextHop, nil
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// padTo rellena un slice a un tamaño exacto.
func padTo(data []byte, size int) []byte {
	out := make([]byte, size)
	copy(out, data)
	return out
}

// min devuelve el mínimo de dos enteros.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// uint32Bytes convierte uint32 a 4 bytes.
func uint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
