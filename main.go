package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/neonxp/rutina"
	"golang.org/x/crypto/acme/autocert"
)

var Hosts []host
var mu sync.Mutex

func main() {
	r := rutina.New(rutina.WithListenOsSignals())
	w, err := newWatcher()
	if err != nil {
		panic(err)
	}
	handler := getHandler(w)
	httpSrv := &http.Server{Addr: ":http", Handler: handler}
	httpsSrv := &http.Server{Addr: ":https", Handler: handler}

	// Docker
	r.Go(w.watch)

	// HTTPS
	r.Go(func(ctx context.Context) error {
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
			Cache:      autocert.DirCache("certs"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(hosts...),
		}
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
		}
		tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpsSrv.Shutdown(tctx)
	}, rutina.RestartIfDone)

	// HTTP
	r.Go(func(ctx context.Context) error {
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		log.Println("Graceful shutdown")
		tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpSrv.Shutdown(tctx)
	})

	if err := r.Wait(); err != nil {
		log.Fatal(err)
	}
	log.Println("Success stop")
}
