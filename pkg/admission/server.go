package admission

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/noobaa/noobaa-operator/v5/pkg/util"
)

const (
	port = "8080"
)

var (
	tlscert, tlskey string
)

// Start admission control server
func RunAdmissionServer() {

	flag.StringVar(&tlscert, "tlsCertFile", "/etc/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&tlskey, "tlsKeyFile", "/etc/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")

	flag.Parse()

	log := util.Logger()
	
	certs, err := tls.LoadX509KeyPair(tlscert, tlskey)
	if err != nil {
		log.Errorf("Filed to load key pair: %v", err)
	}

    if err != nil {
        fmt.Println(err.Error())
        return
    }

	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{certs}},
	}

	// define http server and server handler
	gs := GrumpyServerHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", gs.serve)
	server.Handler = mux

	// start webhook server in new rountine
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	log.Infof("Server running listening in port: %s", port)

	// listening shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info("Got shutdown signal, shutting down webhook server gracefully...")
	server.Shutdown(context.Background())
}