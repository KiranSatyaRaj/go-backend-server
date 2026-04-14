package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) writeNumReqs() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
		w.Header().Add("Content-Type", "text/html")
		if _, err := w.Write([]byte(body)); err != nil {
			fmt.Println(err)
			return
		}
	})
}

func (cfg *apiConfig) resetReqs() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Swap(0)
	})
}

func main() {
	apiConfig := apiConfig{}
	mux := http.NewServeMux()
	mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("/app/assets/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app/assets", http.FileServer(http.Dir("./assets/")))))
	mux.Handle("GET /admin/metrics", apiConfig.writeNumReqs())
	mux.Handle("POST /admin/reset", apiConfig.resetReqs())
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		if _, err := w.Write([]byte("OK")); err != nil {
			fmt.Println(err)
		}
	})
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, req *http.Request) {
		type msgbody struct {
			Body string `json:"body"`
		}
		decoder := json.NewDecoder(req.Body)
		msg := msgbody{}
		type errmsg struct {
			Error string `json:"error"`
		}
		w.Header().Set("Content-Type", "application/json")
		err := decoder.Decode(&msg)
		if err != nil {
			respBody := errmsg{"Something went wrong"}
			dat, _ := json.Marshal(respBody)
			w.WriteHeader(500)
			w.Write(dat)
			return
		}
		if len(msg.Body) > 140 {
			respBody := errmsg{"Chirp is too long"}
			dat, _ := json.Marshal(respBody)
			w.WriteHeader(400)
			w.Write(dat)
			return
		}

		words := strings.Split(msg.Body, " ")
		for i := 0; i < len(words); i++ {
			word := strings.ToLower(words[i])
			if word == "kerfuffle" || word == "sharbert" || word == "fornax" {
				words[i] = "****"
			}
		}
		msg.Body = strings.Join(words, " ")
		type cleanedBody struct {
			Body string `json:"cleaned_body"`
		}

		respBody := cleanedBody{msg.Body}
		dat, _ := json.Marshal(respBody)
		w.WriteHeader(200)
		w.Write(dat)
	})
	server := http.Server{Handler: mux}
	server.Addr = ":8080"
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
