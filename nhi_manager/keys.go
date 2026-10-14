package nhi_manager

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
)

// One bundle commits all three keys together, preventing partial key rotation.
// Keep this directory outside any browsable workspace and back it up securely.
type keyBundle struct {
	Version         int    `json:"version"`
	CACertificate   []byte `json:"ca_certificate"`
	CAPrivateKey    []byte `json:"ca_private_key"`
	JWTKey          []byte `json:"jwt_key"`
	AuditPrivateKey []byte `json:"audit_private_key"`
}
type managerKeys struct {
	ca    *CertAuthority
	jwt   JWTSigningKey
	audit *DilithiumKeyPair
}

func generateManagerKeys() (*managerKeys, error) {
	ca, err := NewCertAuthority()
	if err != nil {
		return nil, err
	}
	jwt, err := GenerateSigningKey()
	if err != nil {
		return nil, err
	}
	audit, err := GenerateDilithiumKeyPair()
	if err != nil {
		return nil, err
	}
	return &managerKeys{ca, jwt, audit}, nil
}
func (k *managerKeys) encode() ([]byte, error) {
	private, err := x509.MarshalPKCS8PrivateKey(k.ca.caKey)
	if err != nil {
		return nil, err
	}
	audit, err := k.audit.PrivateKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return json.Marshal(keyBundle{1, k.ca.caCert.Raw, private, k.jwt, audit})
}
func decodeManagerKeys(raw []byte) (*managerKeys, error) {
	var bundle keyBundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return nil, fmt.Errorf("invalid manager key bundle")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || bundle.Version != 1 || len(bundle.JWTKey) != 32 {
		return nil, fmt.Errorf("unsupported manager key bundle")
	}
	cert, err := x509.ParseCertificate(bundle.CACertificate)
	if err != nil {
		return nil, fmt.Errorf("invalid stored CA certificate")
	}
	private, err := x509.ParsePKCS8PrivateKey(bundle.CAPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid stored CA key")
	}
	key, ok := private.(*rsa.PrivateKey)
	if !ok || key.Validate() != nil || key.N.BitLen() < 4096 || !cert.IsCA || !cert.BasicConstraintsValid || cert.KeyUsage&x509.KeyUsageCertSign == 0 || !key.PublicKey.Equal(cert.PublicKey) || cert.CheckSignatureFrom(cert) != nil || time.Now().Before(cert.NotBefore) || !time.Now().Before(cert.NotAfter) {
		return nil, fmt.Errorf("stored CA trust validation failed")
	}
	audit := new(mode3.PrivateKey)
	if err := audit.UnmarshalBinary(bundle.AuditPrivateKey); err != nil {
		return nil, fmt.Errorf("invalid stored audit key")
	}
	public, ok := audit.Public().(*mode3.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid audit public key")
	}
	ca := &CertAuthority{caCert: cert, caKey: key, caCertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})}
	return &managerKeys{ca, JWTSigningKey(bundle.JWTKey), &DilithiumKeyPair{PublicKey: public, PrivateKey: audit}}, nil
}
func auditKeyID(public []byte) string {
	sum := sha256.Sum256(public)
	return "nexus-audit-" + hex.EncodeToString(sum[:])
}
func readManagerKeys(path string) (*managerKeys, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 64<<10 {
		return nil, fmt.Errorf("manager key file must be private, regular and bounded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("manager key file changed while opening")
	}
	raw, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	if err != nil || len(raw) > 64<<10 {
		return nil, fmt.Errorf("cannot read manager keys")
	}
	return decodeManagerKeys(raw)
}
func loadManagerKeys(dir string) (*managerKeys, error) {
	if dir == "" {
		return generateManagerKeys()
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("manager key directory must be private and must not be a symlink")
	}
	path := filepath.Join(dir, "manager-keys.json")
	if keys, err := readManagerKeys(path); err == nil {
		return keys, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	keys, err := generateManagerKeys()
	if err != nil {
		return nil, err
	}
	raw, err := keys.encode()
	if err != nil {
		return nil, err
	}
	temporary, err := os.CreateTemp(dir, ".manager-keys-*")
	if err != nil {
		return nil, err
	}
	name := temporary.Name()
	defer os.Remove(name)
	defer temporary.Close()
	if _, err := temporary.Write(raw); err != nil {
		return nil, err
	}
	if err := temporary.Sync(); err != nil {
		return nil, err
	}
	if err := temporary.Close(); err != nil {
		return nil, err
	}
	// Hard-link publication is atomic and never overwrites another initializer.
	if err := os.Link(name, path); err != nil {
		if os.IsExist(err) {
			return readManagerKeys(path)
		}
		return nil, err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return nil, err
	}
	return keys, nil
}
