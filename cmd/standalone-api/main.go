package main

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/jonsabados/saturdaysspinout/cmd"
)

// requestTimeout matches the Lambda timeout configured in terraform/api.tf
const requestTimeout = 15 * time.Second

func main() {
	listenAddress := flag.String("listen-address", ":8080", "address to listen to for inbound requests")
	flag.Parse()

	handler := withRequestTimeout(cmd.CreateAPI(), requestTimeout)

	err := http.ListenAndServe(*listenAddress, handler)
	if err != nil {
		panic(err)
	}
}

func withRequestTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
