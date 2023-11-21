package main

import (
	"crypto/tls"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Certificates reload
type keyReloader struct {
	mu       sync.RWMutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
}

func (s *server) newKeyReloader(certPath, keyPath string) (*keyReloader, error) {
	res := &keyReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	res.cert = &cert
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		for range c {
			s.logger.Println("Received SIGUSR1, reloading config")
			if err := res.maybeReload(); err != nil {
				s.logger.Println("Error reloading cert:", err)
			}
		}
	}()
	return res, nil
}

func (kr *keyReloader) maybeReload() error {
	newCert, err := tls.LoadX509KeyPair(kr.certPath, kr.keyPath)
	if err != nil {
		return err
	}
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.cert = &newCert
	return nil
}

func (kr *keyReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		kr.mu.RLock()
		defer kr.mu.RUnlock()
		return kr.cert, nil
	}
}
