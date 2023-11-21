package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type server struct {
	domain   string
	thermKey string
	srv      *http.Server
	router   *http.ServeMux
	logger   *log.Logger
	therm    *therm
	watchdog *watchdog
}

func main() {
	var err error
	s := server{
		domain: "my.example.com",
	}

	// Logger
	s.logger = log.New(os.Stdout, "therm: ", log.LstdFlags)

	// Config
	srvListen, ok := os.LookupEnv("THERMADDR")
	if !ok {
		s.logger.Fatal("Error: missing THERMADDR env config eg. 127.0.0.1:443")
	}
	certDir, ok := os.LookupEnv("THERMCERTDIR")
	// Contains fullchain.pem & privkey.pem
	// cerated with : $ openssl req -x509 -nodes -sha256 -days 365 -newkey rsa:2048 -keyout privkey.pem -out fullchain.pem
	// or with Letsencrypt
	if !ok {
		s.logger.Fatal("Error: missing THERMCERTDIR env config eg. /etc/letsencrypt/live/my.example.com")
	}
	s.thermKey, ok = os.LookupEnv("THERMAUTHAPIKEY")
	if !ok {
		s.logger.Fatal("Error: missing THERMAUTHAPIKEY env config (base64-encoded)")
	}

	// Thermostat
	s.therm, err = NewTherm(nil)
	if err != nil {
		s.logger.Fatal("Error setting-up the thermostat:", err.Error())
	}
	defer s.therm.Close()

	// TLS configuration
	kr, err := s.newKeyReloader(
		certDir+"/fullchain.pem",
		certDir+"/privkey.pem")
	if err != nil {
		s.logger.Fatal("Error creating TLS certificate loader:", err)
	}
	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		GetCertificate: kr.GetCertificateFunc(),
	}

	// Router
	s.router = http.NewServeMux()
	s.routes()

	// Server
	s.srv = &http.Server{
		Addr:         srvListen,
		Handler:      s.logMid(s.router),
		ErrorLog:     s.logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// Handling graceful shutdown
	srvClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		s.logger.Println("Received interrupt, shutting down server...")
		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.logger.Fatal("Error shutting down server:", err)
		}
		s.watchdog.Close() // might cause an issue if didn't have time to launch
		close(srvClosed)
	}()

	// Launching watchdog
	// Revert to safe state if no request for a certain amount of time
	// eg loss of internet connection
	s.watchdog = NewWatchdog(func() {
		s.logger.Println("Watchdog reverted to safe state: have not had any contact for a while")
		s.therm.SetState(safeThermState)
	})

	// Starting server
	s.logger.Println("Starting server:", s.srv)
	if err := s.srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		s.logger.Fatal("Server error:", err)
	}

	// Closing server
	<-srvClosed
	s.logger.Println("Server closed. Bye!")
}
