package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	url    = flag.String("u", "http://192.168.122.1:33333/dl", "URL")
	nConns = flag.Int("n", 1, "# of connections")
)

func main() {
	flag.Parse()
	var wg sync.WaitGroup
	wg.Add(*nConns)
	for i := 0; i < *nConns; i++ {
		go func() {
			defer wg.Done()
			err := dl(*url)
			if err != nil {
				fmt.Printf("download failed: %v\n", err)
			}
		}()
	}
	wg.Wait()
}
