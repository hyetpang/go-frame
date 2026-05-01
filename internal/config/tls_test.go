package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateSelfSignedCert 在临时目录生成自签证书,返回 (certFile, keyFile)。
func generateSelfSignedCert(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成私钥出错: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "go-frame-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("生成证书出错: %v", err)
	}

	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), 0600); err != nil {
		t.Fatalf("写入 cert 出错: %v", err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("序列化私钥出错: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}), 0600); err != nil {
		t.Fatalf("写入 key 出错: %v", err)
	}
	return certFile, keyFile
}

func TestTLSConfigDisabledReturnsNil(t *testing.T) {
	c := &TLSConfig{Enable: false}

	got, err := c.BuildClientTLS()
	if err != nil || got != nil {
		t.Fatalf("BuildClientTLS(disabled) = (%v, %v), want (nil, nil)", got, err)
	}
	got, err = c.BuildServerTLS()
	if err != nil || got != nil {
		t.Fatalf("BuildServerTLS(disabled) = (%v, %v), want (nil, nil)", got, err)
	}
	if c.IsEnabled() {
		t.Fatal("IsEnabled() = true, want false")
	}
}

func TestBuildClientTLSWithoutCertsHappyPath(t *testing.T) {
	c := &TLSConfig{
		Enable:     true,
		ServerName: "example.internal",
	}
	got, err := c.BuildClientTLS()
	if err != nil {
		t.Fatalf("BuildClientTLS error: %v", err)
	}
	if got == nil {
		t.Fatal("BuildClientTLS returned nil tls.Config")
	}
	if got.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %x, want TLS1.2", got.MinVersion)
	}
	if got.ServerName != "example.internal" {
		t.Fatalf("ServerName = %q, want example.internal", got.ServerName)
	}
}

func TestBuildClientTLSCertWithoutKeyFails(t *testing.T) {
	c := &TLSConfig{
		Enable:   true,
		CertFile: "/tmp/no-such-cert.pem",
	}
	if _, err := c.BuildClientTLS(); err == nil {
		t.Fatal("expected cert/key 不成对错误")
	}
}

func TestBuildClientTLSInvalidCAFails(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad-ca.pem")
	if err := os.WriteFile(badCA, []byte("NOT A VALID PEM"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &TLSConfig{Enable: true, CAFile: badCA}
	if _, err := c.BuildClientTLS(); err == nil {
		t.Fatal("expected CA PEM 解析失败错误")
	}
}

func TestBuildClientTLSWithClientCertSucceeds(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateSelfSignedCert(t, dir)

	c := &TLSConfig{
		Enable:   true,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	got, err := c.BuildClientTLS()
	if err != nil {
		t.Fatalf("BuildClientTLS 加载客户端证书出错: %v", err)
	}
	if len(got.Certificates) != 1 {
		t.Fatalf("Certificates len = %d, want 1", len(got.Certificates))
	}
}

func TestBuildServerTLSRequiresCertAndKey(t *testing.T) {
	c := &TLSConfig{Enable: true}
	if _, err := c.BuildServerTLS(); err == nil {
		t.Fatal("expected 缺 cert/key 错误")
	}
}

func TestBuildServerTLSHappyPath(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateSelfSignedCert(t, dir)

	c := &TLSConfig{
		Enable:   true,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	got, err := c.BuildServerTLS()
	if err != nil {
		t.Fatalf("BuildServerTLS error: %v", err)
	}
	if got.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %x, want TLS1.2", got.MinVersion)
	}
	if len(got.Certificates) != 1 {
		t.Fatalf("Certificates len = %d, want 1", len(got.Certificates))
	}
	if got.ClientAuth != tls.NoClientCert {
		t.Fatalf("ClientAuth = %v, want NoClientCert (无 CA 时不强制 mTLS)", got.ClientAuth)
	}
}

func TestBuildServerTLSWithCAEnablesMTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateSelfSignedCert(t, dir)

	c := &TLSConfig{
		Enable:   true,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   certFile, // 用同一份证书当 CA,仅验证 ClientCAs 加载逻辑
	}
	got, err := c.BuildServerTLS()
	if err != nil {
		t.Fatalf("BuildServerTLS error: %v", err)
	}
	if got.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatalf("ClientAuth = %v, want RequireAndVerifyClientCert", got.ClientAuth)
	}
	if got.ClientCAs == nil {
		t.Fatal("ClientCAs 未加载")
	}
}
