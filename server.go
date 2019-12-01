package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func getHandler(watcher *watcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostInfo, loaded := watcher.Load(r.Host)
		if !loaded {
			w.WriteHeader(404)
			return
		}
		h := hostInfo.(host)
		if h.TLS {
			newUrl := *r.URL
			newUrl.Scheme = "https"
			http.Redirect(w, r, newUrl.String(), http.StatusPermanentRedirect)
		}
		remoteUrl := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", h.Addr, h.Port),
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		proxy.ServeHTTP(w, r)
		return
	})
}

func getTlsHandler(watcher *watcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostInfo, loaded := watcher.Load(r.Host)
		if !loaded {
			w.WriteHeader(404)
			return
		}
		h := hostInfo.(host)
		if !h.TLS {
			w.WriteHeader(404)
			return
		}
		remoteUrl := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", h.Addr, h.Port),
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		proxy.ServeHTTP(w, r)
		return
	})
}
