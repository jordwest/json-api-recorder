package main

import (
	"fmt"
	"net/http"
	"time"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got a request to /login")
	fmt.Fprintf(w, "Hello!")
}

func main() {
	/*
		localProxy, err := interceptRequest("https://192.0.78.23", "public-api.wordpress.com")
		if err != nil {
			fmt.Println("Error setting up reverse proxy", err)
		}
	*/

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/", handleRequest)

	fmt.Println("HTTPS listening on :443")
	go func() {
		err := http.ListenAndServeTLS(":443", "./cert.pem", "./key.pem", nil)
		time.Sleep(time.Second)
		fmt.Println("Listener on :443 failed:", err)
	}()

	fmt.Println("HTTP listening on :80")
	go func() {
		err := http.ListenAndServe(":80", nil)
		time.Sleep(time.Second)
		fmt.Println("Listener on :80 failed:", err)
	}()

	for {
		time.Sleep(time.Second)
	}
}
