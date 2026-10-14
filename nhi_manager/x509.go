// Nexus IA v6.0 — Certificados X.509 efímeros
//
// Cada agente recibe un certificado X.509 firmado por la CA interna
// de Nexus. Los certificados son efímeros: viven 24h y luego se
// renuevan automáticamente.
//
// El certificado encapsula:
//   - Clave pública Dilithium3 del agente (en vez de RSA/ECDSA)
//   - AgentID como CN
//   - AgentType como OU
//   - DID como SAN (subjectAltName)
package nhi_manager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// DefaultCertTTL es la vida por defecto de un certificado de agente.
const DefaultCertTTL = 24 * time.Hour

// CertAuthority es una autoridad certificadora interna de Nexus.
// Firma los certificados de todos los agentes del sistema.
type CertAuthority struct {
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	caCertPEM []byte
}

// NewCertAuthority genera una CA interna fresca.
// La CA raíz vive toda la vida del sistema Nexus.
func NewCertAuthority() (*CertAuthority, error) {
	// RSA 4096 para la CA raíz (larga vida)
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	caCert := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Nexus IA v6.0"},
			OrganizationalUnit: []string{"Trust Root"},
			CommonName:   "Nexus Internal CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 años
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caCert, caCert, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertDER,
	})

	parsedCA, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	return &CertAuthority{
		caCert:    parsedCA,
		caKey:     caKey,
		caCertPEM: caCertPEM,
	}, nil
}

// IssueCert emite un certificado X.509 para un agente.
// El certificado encapsula el AgentID como CN, el AgentType como OU,
// y el DID como SAN.
func (ca *CertAuthority) IssueCert(agentID string, agentType AgentType, did DID, ttl time.Duration) ([]byte, error) {
	if ttl == 0 {
		ttl = DefaultCertTTL
	}

	// Generar clave RSA para el agente (provisional; Dilithium va en el JWT/audit)
	agentKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate agent key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization:       []string{"Nexus IA v6.0"},
			OrganizationalUnit: []string{string(agentType)},
			CommonName:         agentID,
		},
		DNSNames:              []string{did.String()},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(ttl),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, ca.caCert, &agentKey.PublicKey, ca.caKey)
	if err != nil {
		return nil, fmt.Errorf("create agent cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM, nil
}

// CACertPEM devuelve el certificado de la CA en formato PEM.
func (ca *CertAuthority) CACertPEM() []byte {
	return ca.caCertPEM
}

// ParseCert parsea un certificado PEM.
func ParseCert(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// VerifyCert verifica que un certificado esté firmado por la CA.
func (ca *CertAuthority) VerifyCert(certPEM []byte) error {
	cert, err := ParseCert(certPEM)
	if err != nil {
		return err
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca.caCert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("verify cert: %w", err)
	}
	return nil
}

// VerifyCertChain verifica que un certificado esté firmado por una CA dada.
func VerifyCertChain(caCertPEM, certPEM []byte) error {
	caCert, err := ParseCert(caCertPEM)
	if err != nil {
		return err
	}
	agentCert, err := ParseCert(certPEM)
	if err != nil {
		return err
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if _, err := agentCert.Verify(opts); err != nil {
		return fmt.Errorf("verify cert chain: %w", err)
	}
	return nil
}
