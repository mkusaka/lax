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
	server *http.Server
	useTLS bool
	// channel for manage all server goroutine error.
	FinishChan chan error
	logger     *log.Logger
	certFile   string
	keyFile    string
}

/*
 * Functions
 */
func Handler(pathStr string, handler http.Handler) {
	http.Handle(pathStr, handler)
}

func (httpServer *HttpServer) ListenAndServe() {
	var err error

	httpServer.logger.Printf("served at %v\n", httpServer.server.Addr)
	if httpServer.useTLS {
		err = httpServer.server.ListenAndServeTLS(httpServer.certFile, httpServer.keyFile)
	} else {
		err = httpServer.server.ListenAndServe()
	}
	httpServer.logger.Printf("Http server goroutine stoped with error. %v\n", err)
	httpServer.FinishChan <- err
}

func NewHttpServer(
	addr string,
	port uint,
	readTimeoutSec time.Duration,
	writeTimeoutSec time.Duration,
	maxHeaderBytes int,
	finishChan chan error,
) *HttpServer {
	addrAndPort := fmt.Sprintf("%v:%v", addr, port)

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	server := &http.Server{
		Addr:           addrAndPort,
		ReadTimeout:    readTimeoutSec * time.Second,
		WriteTimeout:   writeTimeoutSec * time.Second,
		MaxHeaderBytes: maxHeaderBytes,
		ErrorLog:       logger,
	}

	httpServer := &HttpServer{
		server:     server,
		logger:     logger,
		FinishChan: finishChan,
		useTLS:     false,
	}

	return httpServer
}

func NewHttpsServer(
	addr string,
	port uint,
	readTimeoutSec time.Duration,
	writeTimeoutSec time.Duration,
	maxHeaderBytes int,
	finishChan chan error,
	certFilePath string,
	keyFilePath string,
) *HttpServer {
	addrAndPort := fmt.Sprintf("%v:%v", addr, port)

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	server := &http.Server{
		Addr:           addrAndPort,
		ReadTimeout:    readTimeoutSec * time.Second,
		WriteTimeout:   writeTimeoutSec * time.Second,
		MaxHeaderBytes: maxHeaderBytes,
		ErrorLog:       logger,
	}

	httpServer := &HttpServer{
		server:     server,
		logger:     logger,
		FinishChan: finishChan,
		certFile:   certFilePath,
		keyFile:    keyFilePath,
		useTLS:     true,
	}

	return httpServer
}
