package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/neonxp/rutina"
	"golang.org/x/crypto/acme/autocert"
)

var httpSrv *http.Server
var httpsSrv *http.Server

func main() {
	certDir := os.Getenv("CERTDIR")
	if certDir == "" {
		certDir = "/usr/app/certs"
	}
	r := rutina.New(rutina.WithListenOsSignals())
	w, err := newWatcher()
	if err != nil {
		panic(err)
	}

	// Docker
	r.Go(w.watch)

	// HTTPS
	r.Go(func(ctx context.Context) error {
		httpsSrv = &http.Server{Addr: ":https", Handler: getTlsHandler(w)}
		hosts := []string{}
		w.Range(func(key, value interface{}) bool {
			h := value.(host)
			if !h.TLS {
				return true
			}
			hosts = append(hosts, h.Host)
			return true
		})
		m := &autocert.Manager{
			Cache:      autocert.DirCache(certDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(hosts...),
		}
		log.Println("https hosts:", hosts)
		httpsSrv.TLSConfig = m.TLSConfig()
		if err := httpsSrv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			return err
		}
		return nil
	}, rutina.RestartIfDone)
	r.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case <-w.update():
			log.Println("reload https config")
		}
		if httpsSrv == nil {
			return nil
		}
		tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpsSrv.Shutdown(tctx)
	}, rutina.RestartIfDone)

	// HTTP
	r.Go(func(ctx context.Context) error {
		httpSrv = &http.Server{Addr: ":http", Handler: getHandler(w)}
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		log.Println("Graceful shutdown")
		if httpsSrv == nil {
			return nil
		}
		tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(tctx)
	})

	if err := r.Wait(); err != nil {
		log.Fatal(err)
	}
	log.Println("Success stop")
}
