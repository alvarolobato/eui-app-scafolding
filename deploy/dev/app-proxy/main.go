package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func main() {
	frontendURL := &url.URL{Scheme: "http", Host: "app-frontend:3000"}
	backendURL := &url.URL{Scheme: "http", Host: "app-backend:4000"}
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			if strings.HasPrefix(pr.In.URL.Path, "/api/") {
				pr.SetURL(backendURL)
			} else {
				pr.SetURL(frontendURL)
			}
			pr.SetXForwarded()
		},
	}
	if err := http.ListenAndServeTLS(":8443", "/tls/cert.pem", "/tls/key.pem", rp); err != nil {
		log.Fatal(err)
	}
}
