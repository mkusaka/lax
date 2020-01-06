package http_server_wrapper

/*
 * Module Dependencies
 */

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

/*
 * Types
 */

type HttpServer struct {
	server     *http.Server
	FinishChan chan error
	logger     *log.Logger
}

/*
 * Constants
 */

/*
 * Functions
 */

func AddHandler(pathStr string, handler http.Handler) {
	http.Handle(pathStr, handler)
}

func (httpServer *HttpServer) ListenAndServe() {
	err := httpServer.server.ListenAndServe()
	httpServer.FinishChan <- err
}

func (httpsServer *HttpServer) ListenAndServeTLS(certFile string, keyFile string) {
	err := httpsServer.server.ListenAndServeTLS(certFile, keyFile)
	httpsServer.FinishChan <- err
}

func NewHttpServer(
	addrStr string,
	port uint,
	readTimeoutSec time.Duration,
	writeTimeoutSec time.Duration,
	maxHeaderBytes int,
) *HttpServer {
	// Join addrStr and port by ":"
	addrAndPortStr := fmt.Sprintf("%v:%v", addrStr, port)

	// Create pointer of HttpServer structure.
	httpServer := new(HttpServer)

	// Create pointer of inner http.Server
	httpServer.server = &http.Server{
		Addr:           addrAndPortStr,
		ReadTimeout:    readTimeoutSec * time.Second,
		WriteTimeout:   writeTimeoutSec * time.Second,
		MaxHeaderBytes: maxHeaderBytes,
	}

	// Create channel
	httpServer.FinishChan = make(chan error)

	// Create logger
	httpServer.logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	httpServer.server.ErrorLog = httpServer.logger

	return httpServer
}
