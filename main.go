package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mkusaka/lax/client"
	"github.com/mkusaka/lax/http_server_wrapper"
)

func main() {
	// TODO: allow auto restart like HUP USR2, or http request like /restart
	var addr string = ""
	var portHttp uint = 10080
	var portHttps uint = 10443
	var readTimeOut time.Duration = 10
	var writeTimeOut time.Duration = 10
	var maxHeaderSize int = 1 << 20
	var keyFilePath string = "./key.pem"
	var certFilePath string = "./cert.pem"

	httpServer := http_server_wrapper.NewHttpServer(
		addr,
		portHttp,
		readTimeOut,
		writeTimeOut,
		maxHeaderSize,
	)
	httpsServer := http_server_wrapper.NewHttpServer(
		addr,
		portHttps,
		readTimeOut,
		writeTimeOut,
		maxHeaderSize,
	)
	http_server_wrapper.AddHandler("/", http.HandlerFunc(proxyHander))

	go httpServer.ListenAndServe()
	fmt.Printf("served at localhost:%v\n", portHttp)

	go httpsServer.ListenAndServeTLS(certFilePath, keyFilePath)
	fmt.Printf("served at localhost:%v\n", portHttps)

	select {
	case httpServerErr := <-httpServer.FinishChan:
		if httpServerErr != nil {
			fmt.Printf(
				"Http server goroutine stoped with error. %v\n",
				httpServerErr)
			os.Exit(1)
		}
	case httpsServerErr := <-httpsServer.FinishChan:
		if httpsServerErr != nil {
			fmt.Printf(
				"Https server goroutine stoped with error. %v\n",
				httpsServerErr)
			os.Exit(1)
		}
	}
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
