// Generate private key (.key)
//
// Key considerations for algorithm "RSA" ≥ 2048-bit openssl genrsa -out
// server.key 2048
//
// Key considerations for algorithm "ECDSA" (X25519 || ≥ secp384r1)
// https://safecurves.cr.yp.to/ # List ECDSA the supported curves (openssl
// ecparam -list_curves) openssl ecparam -genkey -name secp384r1 -out
// server.key
//
// Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based
// on the private (.key) openssl req -new -x509 -sha256 -key server.key -out
// server.crt -days 3650
//
// See: https://github.com/denji/golang-tls
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

func main() {
	http.HandleFunc("/", handle)
	http.HandleFunc("/dl", limitWithStats(5*GiB, dl))
	http.HandleFunc("/dlz", dlz)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		http.ListenAndServe("0.0.0.0:33333", nil)
	}()
	http.ListenAndServeTLS("0.0.0.0:33433", "server.crt", "server.key", nil)
	wg.Wait()
}

func handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Hello TLS server!\n")
}

var (
	zeroOctets      [2 << 20]byte
	zeroOctetsShort [2 << 19]byte
)

func dl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	for {
		_, err := w.Write(zeroOctets[:])
		if err != nil {
			break
		}
	}
}

func dlz(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("/dev/zero")
	if err != nil {
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = io.Copy(w, f)
	if err != nil {
		fmt.Printf("io copy: %v\n", err)
	}
}

// Common size suffixes
const (
	B   = 1
	KiB = 1 << (10 * iota)
	MiB
	GiB
	TiB
	PiB
	EiB
)

type statResponseWriter struct {
	http.ResponseWriter
	nbytes int64
}

func (s *statResponseWriter) Write(p []byte) (n int, err error) {
	n, err = s.ResponseWriter.Write(p)
	s.nbytes += int64(n)
	return n, err
}

type limitedResponseWriter struct {
	http.ResponseWriter       // underlying writer
	n                   int64 // max bytes remaining
}

func (l *limitedResponseWriter) Write(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err = l.ResponseWriter.Write(p)
	l.n -= int64(n)
	return
}

func limit(nbytes int64, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lw := &limitedResponseWriter{
			ResponseWriter: w,
			n:              nbytes,
		}
		h(lw, r)
	}
}

func stats(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(&statResponseWriter{ResponseWriter: w}, r)
	}
}

func limitWithStats(nbytes int64, h http.HandlerFunc) http.HandlerFunc {
	lw := limit(nbytes, h)
	sw := stats(lw)
	return sw
}
