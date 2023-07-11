package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	surl "net/url"
	"sync"
)

func dl(url string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

var (
	url         = flag.String("u", "http://192.168.122.1:33333", "URL")
	nConns      = flag.Int("n", 1, "# of connections")
	nShortLived = flag.Int("nshort", 0, "# of short lived connections")
)

const (
	dlPath    = "dl"
	shortPath = "dlsr"

	maxErrors = 10
)

func main() {
	flag.Parse()
	dst, err := surl.JoinPath(*url, dlPath)
	if err != nil {
		log.Fatalf("failed to join url: %v", err)
	}
	shortDst, err := surl.JoinPath(*url, shortPath)
	if err != nil {
		log.Fatalf("failed to join url: %v", err)
	}
	semC := make(chan struct{}, *nConns+*nShortLived)
	var numErrors int
	var wg sync.WaitGroup
	wg.Add(*nConns)
	for i := 0; i < *nConns && numErrors < maxErrors; i++ {
		semC <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-semC
			}()
			err := dl(dst)
			if err != nil {
				fmt.Printf("download failed: %v\n", err)
				numErrors++
			}
		}()
	}
	for numErrors < maxErrors {
		wg.Add(1)
		semC <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-semC
			}()
			err := dl(shortDst)
			if err != nil {
				fmt.Printf("download failed: %v\n", err)
				numErrors++
			}
		}()
	}
	wg.Wait()
}
