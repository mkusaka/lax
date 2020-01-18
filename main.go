package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/mkusaka/lax/client"
	"github.com/mkusaka/lax/http_server_wrapper"
)

func main() {
	// TODO: allow auto restart like HUP USR2, or http request like /restart
	addr := ""
	portHttp := uint(10080)
	portHttps := uint(10443)
	readTimeOut := time.Duration(10)
	writeTimeOut := time.Duration(10)
	// 1 << 20 from https://golang.org/pkg/net/http/
	maxHeaderSize := 1 << 20
	keyFilePath := "./key.pem"
	certFilePath := "./cert.pem"

	serverErrorChan := make(chan error)
	httpServer := http_server_wrapper.NewHttpServer(
		addr,
		portHttp,
		readTimeOut,
		writeTimeOut,
		maxHeaderSize,
		serverErrorChan,
	)

	httpsServer := http_server_wrapper.NewHttpsServer(
		addr,
		portHttps,
		readTimeOut,
		writeTimeOut,
		maxHeaderSize,
		serverErrorChan,
		certFilePath,
		keyFilePath,
	)
	http_server_wrapper.Handler("/", http.HandlerFunc(proxyHander))

	go httpServer.ListenAndServe()

	go httpsServer.ListenAndServe()
	// TODO: graceful shutdown
	// if http or https server crashed, then the other server have to graceful shutdown
	<-serverErrorChan
}

func proxyHander(w http.ResponseWriter, r *http.Request) {
	c := client.NewClient(10 * time.Second)
	// TODO: not ruled pattern makes panic
	res, err := c.ProxyRequest(r)

	if err != nil {
		// TODO: log store from worker
		// TODO: error handling. add retry function
		// TODO: send error report to primary server
		http.Error(w, http.StatusText(http.StatusBadRequest)+" :"+err.Error(), http.StatusBadRequest)
		return
	}

	// res.Header returns map[string][]string
	// TODO: following code valid? (write all header from origin server to response from proxy server)
	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)

	if res.Body != nil {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("invalid encoding: %s", err)
		}
		w.Write(body)
	}
}
