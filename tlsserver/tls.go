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
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/go-units"
)

var (
	verbose = flag.Bool("v", false, "verbose output")
)

func main() {
	flag.Parse()
	var wg sync.WaitGroup
	srv := newServer()
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := http.ListenAndServe("0.0.0.0:33333", srv)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	go func() {
		err := http.ListenAndServeTLS("0.0.0.0:33433", "server.crt", "server.key", srv)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	tick := time.Tick(1 * time.Second)
	var prev uint64
	for range tick {
		b := srv.nsent.Load()
		fmt.Printf("                                          \r")
		fmt.Printf("bandwidth: %v\t%s/s\r", bytes(b), bitsSize((b-prev)*ToBits))
		prev = b
	}
	wg.Wait()
}

func newServer() *server {
	srv := &server{}
	srv.routes()
	return srv
}

type server struct {
	// Total number bytes sent to the clients
	nsent atomic.Uint64

	mux http.ServeMux
}

func (s *server) routes() {
	s.mux.HandleFunc("/", s.handle)
	s.mux.HandleFunc("/dl", stats(&s.nsent, s.dl))
	s.mux.HandleFunc("/dls", limitWithStats(&s.nsent, 5*GiB, s.dls))
	s.mux.HandleFunc("/dlsr", limitRandBetweenStats(&s.nsent, 3*GiB, 20*GiB, s.dls))
	s.mux.HandleFunc("/dlz", s.dlz)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Hello TLS server!\n")
}

var (
	zeroOctets      [2 << 20]byte
	zeroOctetsShort [2 << 19]byte
)

func (s *server) dl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	for {
		_, err := w.Write(zeroOctets[:])
		if err != nil {
			break
		}
	}
}

// short running conn
func (s *server) dls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	for {
		_, err := w.Write(zeroOctetsShort[:])
		if err != nil {
			break
		}
	}
}

func (s *server) dlz(w http.ResponseWriter, r *http.Request) {
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

type bytes int64

// Common size suffixes
const (
	B   bytes = 1
	KiB bytes = 1 << (10 * iota)
	MiB
	GiB
	TiB
	PiB
	EiB
)

func (b bytes) String() string {
	return units.BytesSize(float64(b))
}

type statResponseWriter struct {
	http.ResponseWriter
	nbytes int64
	stat   *atomic.Uint64
}

func (s *statResponseWriter) Write(p []byte) (n int, err error) {
	n, err = s.ResponseWriter.Write(p)
	s.nbytes += int64(n)
	s.stat.Add(uint64(n))
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

func limit(nbytes bytes, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lw := &limitedResponseWriter{
			ResponseWriter: w,
			n:              int64(nbytes),
		}
		h(lw, r)
	}
}

func limitRandBetween(from, to bytes, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nn := rand.Intn(int(to-from)+1) + int(from)
		if *verbose {
			fmt.Printf("\nwill serve %s\n", bytes(nn))
		}
		lw := &limitedResponseWriter{
			ResponseWriter: w,
			n:              int64(nn),
		}
		h(lw, r)
	}
}

func limitRandBetweenStats(
	stat *atomic.Uint64,
	from, to bytes,
	h http.HandlerFunc,
) http.HandlerFunc {
	sw := stats(stat, h)
	return limitRandBetween(from, to, sw)
}

func stats(stat *atomic.Uint64, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(&statResponseWriter{ResponseWriter: w, stat: stat}, r)
	}
}

func limitWithStats(stat *atomic.Uint64, nbytes bytes, h http.HandlerFunc) http.HandlerFunc {
	lw := limit(nbytes, h)
	sw := stats(stat, lw)
	return sw
}
