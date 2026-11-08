package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), r.Method, "http://localhost:9002"+r.URL.RequestURI(), r.Body)
		if err != nil {
			log.Printf("failed to create request: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.ContentLength = r.ContentLength

		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			log.Printf("backend error: %v", err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)

		io.Copy(w, resp.Body)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}
