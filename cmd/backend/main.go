package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	port := flag.Int("port", 9001, "port to listen on")
	name := flag.String("name", "b1", "backend name in responses and logs")
	delay := flag.Duration("delay", 0, "artificial response delay, e.g. 50ms or 2s")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logRequest(*name, r)

		if !wait(r.Context(), *delay) {
			log.Printf("[%s] %s %s: canceled (%v)", *name, r.Method, r.URL.RequestURI(), r.Context().Err())
			return
		}

		w.Header().Set("X-Backend", *name)
		fmt.Fprintf(w, "Hello from %s\n", *name)
	})

	http.HandleFunc("/status/{code}", func(w http.ResponseWriter, r *http.Request) {
		logRequest(*name, r)
		w.Header().Set("X-Backend", *name)

		if !wait(r.Context(), *delay) {
			log.Printf("[%s] %s %s: canceled (%v)", *name, r.Method, r.URL.RequestURI(), r.Context().Err())
			return
		}

		codeStr := r.PathValue("code")

		code, err := strconv.Atoi(codeStr)
		if err != nil || code < 200 || code > 599 {
			http.Error(w, "Invalid status code", http.StatusBadRequest)
			return
		}

		w.WriteHeader(code)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("backend %s listening on %s, delay %s", *name, addr, *delay)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func logRequest(name string, r *http.Request) {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s %s from %s\n", name, r.Method, r.URL.RequestURI(), r.RemoteAddr)
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(&b, "    %s: %s\n", key, value)
		}
	}
	log.Print(b.String())
}

func wait(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
