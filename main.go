package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.Handle("/app/assets/", http.StripPrefix("/app/assets", http.FileServer(http.Dir("./assets/"))))
	mux.HandleFunc("/healthz/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	server := http.Server{Handler: mux}
	server.Addr = ":8080"
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
