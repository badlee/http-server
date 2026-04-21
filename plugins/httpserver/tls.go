package httpserver

import (
	"beba/plugins/config"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// GetTLSConfig returns a tls.Config based on AppConfig.
// It prioritizes provided Cert/Key files, then ACME (Let's Encrypt),
// and finally falls back to a self-signed certificate.
func GetTLSConfig(cfg *config.AppConfig) (*tls.Config, error) {
	// 1. Manually provided certificate
	if cfg.Cert != "" && cfg.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err == nil {
			return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
		}
		fmt.Printf("Warning: failed to load provided certificate: %v. Falling back...\n", err)
	}

	// 2. Let's Encrypt (ACME)
	if cfg.Domain != "" {
		fmt.Printf("Attempting Let's Encrypt for domain: %s\n", cfg.Domain)
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(cfg.Domain),
			Cache:      autocert.DirCache("cache-certs"),
		}
		if cfg.Email != "" {
			m.Email = cfg.Email
		}
		// Test if it works by getting the TLS config
		// Note: autocert only works on port 443 for the challenge
		return m.TLSConfig(), nil
	}

	// 3. Fallback to self-signed
	fmt.Println("No certificate provided and no domain for ACME. Generating self-signed certificate...")
	cert, err := GenerateSelfSigned()
	if err != nil {
		return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

// GenerateSelfSigned creates a temporary self-signed certificate.
func GenerateSelfSigned() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Beba Auto-Generated"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// SetupACMEForVhost returns a tls.Config for a specific domain using ACME.
func SetupACMEForVhost(domain, email string) *tls.Config {
	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Email:      email,
		Cache:      autocert.DirCache(filepath.Join("cache-certs", domain)),
	}
	return m.TLSConfig()
}
