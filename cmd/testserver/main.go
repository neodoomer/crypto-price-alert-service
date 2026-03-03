package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	secret := os.Getenv("SECRET")

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR reading body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		fmt.Println("========== WEBHOOK RECEIVED ==========")
		fmt.Printf("Time:   %s\n", time.Now().Format(time.RFC3339))
		fmt.Printf("Method: %s\n", r.Method)
		fmt.Println("Headers:")
		for k, v := range r.Header {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Printf("Body:   %s\n", string(body))

		if secret != "" {
			sig := r.Header.Get("X-Signature-256")
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			if hmac.Equal([]byte(sig), []byte(expected)) {
				fmt.Println("Signature: VALID")
			} else {
				fmt.Println("Signature: INVALID")
				fmt.Printf("  Expected: %s\n", expected)
				fmt.Printf("  Got:      %s\n", sig)
			}
		}
		fmt.Println("=======================================")
		fmt.Println()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"received"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	addr := ":" + port
	log.Printf("Test webhook server listening on %s", addr)
	if secret != "" {
		log.Printf("HMAC verification enabled (SECRET is set)")
	}
	log.Fatal(http.ListenAndServe(addr, nil))
}
