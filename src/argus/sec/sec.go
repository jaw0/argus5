// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-18 20:41 (EDT)
// Function: pki security

package sec

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"math/big"
	"time"

	"argus/argus"
	"argus/config"
	"argus/diag"
	"argus/sched"
)

// if we generate a self-signed cert:
const (
	KEYTYPE = "ec256" // ec384, rsa2048, rsa4096, rsa8192
	WEEK    = 7 * 24 * 3600 * time.Second
)

var dl = diag.Logger("sec")

var (
	Root *x509.CertPool
	Cert *tls.Certificate
)

func Init(root string) {

	cf := config.Cf()

	if root == "" {
		root = cf.DARP_root
	}

	loadRoot(root)

	if cf.DARP_key != "" && cf.DARP_cert != "" {
		cert, err := tls.LoadX509KeyPair(cf.DARP_cert, cf.DARP_key)

		if err != nil {
			dl.Fatal("cannot load cert/key: %v", err)
		}
		Cert = &cert
		dl.Debug("keypair loaded")

		CertExpiresWarn(cf.DARP_cert, &cert, cf.DARP_Name)
	}

	if Cert == nil {
		generateCert()
		dl.Debug("keypair generated")
	}
}

func loadRoot(file string) {

	if file == "" {
		return
	}

	roots := x509.NewCertPool()

	data, err := ioutil.ReadFile(file)
	if err != nil {
		dl.Fatal("cannot open root cert '%s': %v", file, err)
	}

	ok := roots.AppendCertsFromPEM(data)
	if !ok {
		dl.Fatal("invalid root cert '%s'", file)
	}

	dl.Debug("root cert loaded")

	Root = roots
}

func CertFileExpiresWarn(fcert, fkey string) {

	cert, err := tls.LoadX509KeyPair(fcert, fkey)
	if err != nil {
		return
	}
	CertExpiresWarn(fcert, &cert, "")
}

func CertExpiresWarn(file string, cert *tls.Certificate, chkname string) {

	x, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}

	now := time.Now()
	expire := x.NotAfter

	// is this the correct cert?
	if chkname != "" {
		name := x.Subject.CommonName

		if name != chkname {
			dl.Fatal("cert name mismatch %s != %s, '%s'", name, chkname, file)
		}
	}

	// is it expired?
	dl.Debug("cert %s expires %s", file, expire.Format("2006-01-02 15:04"))

	if expire.Before(now) {
		dl.Problem("cert expired! '%s'", file)
	}

	// does it expire soon?
	if expire.Add(1 * WEEK).Before(now) {
		argus.ConfigWarning(file, 0, "cert expires soon %s", expire.Format("2006-01-02 15:04"))
		return
	}

	// schedule a warning for later
	when := expire.Add(-WEEK)
	sched.At(when.Unix(), "cert expire", func() {
		argus.ConfigWarning(file, 0, "cert expires soon %s", expire.Format("2006-01-02 15:04"))
	})

	sched.At(expire.Unix(), "cert expire", func() {
		dl.Problem("cert expired! '%s'", file)
		argus.ConfigError(file, 0, "cert expired!")
	})
}

// generate self-signed key pair
func generateCert() {

	var privKey interface{}
	var pubKey interface{}
	var err error

	switch KEYTYPE {
	case "rsa2048":
		privKey, pubKey, err = rsaKey(2048)
	case "rsa4096":
		privKey, pubKey, err = rsaKey(4096)
	case "rsa8192":
		privKey, pubKey, err = rsaKey(8192)
	case "ec256":
		privKey, pubKey, err = ecdsaKey(elliptic.P256())
	case "ec384":
		privKey, pubKey, err = ecdsaKey(elliptic.P384())
	default:
		dl.Bug("invalid key type")
	}

	if err != nil {
		dl.Fatal("cannot generate private key: %v", err)
	}

	now := time.Now()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		dl.Fatal("cannot generate random serial number: %v", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{"Argus Auto-Generated Cert"}},
		NotBefore:    now,
		NotAfter:     now.Add(25 * 365 * 24 * 3600 * time.Second),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, pubKey, privKey)
	if err != nil {
		dl.Fatal("cannot create cert: %v", err)
	}

	Cert = &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privKey,
		Leaf:        cert,
	}

	dl.Verbose("created cert %X", serialNumber)
}

func rsaKey(size int) (interface{}, interface{}, error) {

	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	return privKey, &privKey.PublicKey, nil
}
func ecdsaKey(ec elliptic.Curve) (interface{}, interface{}, error) {

	privKey, err := ecdsa.GenerateKey(ec, rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privKey, &privKey.PublicKey, nil
}
