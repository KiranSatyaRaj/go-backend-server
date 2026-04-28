package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/KiranSatyaRaj/go-backend-server/internal/auth"
	"github.com/KiranSatyaRaj/go-backend-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	tokenSecret    string
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
		if cfg.platform != "dev" {
			w.WriteHeader(401)
		} else {
			cfg.db.DeleteUser(req.Context())
			cfg.fileserverHits.Swap(0)
		}
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	dbQueries := database.New(db)
	platform := os.Getenv("PLATFORM")
	apiConfig := apiConfig{db: dbQueries, platform: platform, tokenSecret: os.Getenv("SECRET")}
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
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, req *http.Request) {
		type msgbody struct {
			Body string `json:"body"`
		}
		decoder := json.NewDecoder(req.Body)
		bearerToken, err := auth.GetBearerToken(req.Header)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		userID, err := auth.ValidateJwt(bearerToken, apiConfig.tokenSecret)
		if err != nil {
			fmt.Println(err, "from chirps")
			w.WriteHeader(401)
			return
		}
		msg := msgbody{}
		type errmsg struct {
			Error string `json:"error"`
		}
		w.Header().Set("Content-Type", "application/json")
		err = decoder.Decode(&msg)
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
		cp := database.CreateChirpParams{msg.Body, userID}
		userChirp, err := apiConfig.db.CreateChirp(req.Context(), cp)
		if err != nil {
			fmt.Println(err)
			return
		}
		respBody := struct {
			Id        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}{
			Id:        userChirp.ID,
			CreatedAt: userChirp.CreatedAt,
			UpdatedAt: userChirp.UpdatedAt,
			Body:      userChirp.Body,
			UserID:    userChirp.UserID,
		}

		dat, _ := json.Marshal(respBody)
		w.WriteHeader(201)
		w.Write(dat)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type AllUserChirps struct {
			Id        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}
		userchirps, _ := apiConfig.db.GetAllUserChirps(r.Context())
		resp := make([]AllUserChirps, len(userchirps))
		for i := 0; i < len(userchirps); i++ {
			resp[i].Body = userchirps[i].Body
			resp[i].Id = userchirps[i].ID
			resp[i].CreatedAt = userchirps[i].CreatedAt
			resp[i].UpdatedAt = userchirps[i].UpdatedAt
			resp[i].UserID = userchirps[i].UserID
		}
		dat, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(dat)
	})
	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			fmt.Println(err)
			return
		}
		chirp, err := apiConfig.db.GetChirpByID(r.Context(), chirpID)
		if err != nil || reflect.ValueOf(chirp).IsZero() {
			w.WriteHeader(404)
			return
		}
		resp := struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		dat, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(dat)
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type userinfo struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		info := userinfo{}
		err := decoder.Decode(&info)
		if err != nil {
			fmt.Println(err)
			return
		}
		hashed_password, err := auth.HashPassword(info.Password)
		if err != nil {
			fmt.Println(err)
			return
		}
		args := database.CreateUserParams{info.Email, hashed_password}
		user, err := apiConfig.db.CreateUser(r.Context(), args)
		resp := struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		}
		w.WriteHeader(201)
		dat, _ := json.Marshal(resp)
		w.Write(dat)
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		loginInfo := struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&loginInfo)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash, _ := apiConfig.db.GetPasswordHash(r.Context(), loginInfo.Email)
		match, err := auth.CheckPasswordHash(loginInfo.Password, hash)
		if err != nil || !match {
			w.WriteHeader(401)
			w.Write([]byte("Incorrect email or password"))
			return
		}
		userinfo, _ := apiConfig.db.GetUserInfo(r.Context(), loginInfo.Email)
		access_token, _ := auth.MakeJwt(userinfo.ID, apiConfig.tokenSecret, time.Duration(1)*time.Hour)
		refresh_token := auth.MakeRefreshToken()
		refresh_token_args := database.CreateRefreshTokenParams{refresh_token, userinfo.ID}
		err = apiConfig.db.CreateRefreshToken(r.Context(), refresh_token_args)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		resp := struct {
			ID           uuid.UUID `json:"id"`
			CreatedAt    time.Time `json:"created_at"`
			UpdatedAt    time.Time `json:"updated_at"`
			Email        string    `json:"email"`
			Token        string    `json:"token"`
			RefreshToken string    `json:"refresh_token"`
		}{
			ID:           userinfo.ID,
			CreatedAt:    userinfo.CreatedAt,
			UpdatedAt:    userinfo.UpdatedAt,
			Email:        userinfo.Email,
			Token:        access_token,
			RefreshToken: refresh_token,
		}
		dat, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(dat)
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		refresh_token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		user_info, err := apiConfig.db.GetUserFromRefreshToken(r.Context(), refresh_token)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		fmt.Println(time.Now(), user_info.RevokedAt.Time)
		if time.Now().UTC().After(user_info.ExpiresAt) || (user_info.RevokedAt.Valid) {
			w.WriteHeader(401)
			return
		}

		access_token, err := auth.MakeJwt(user_info.UserID, apiConfig.tokenSecret, time.Duration(1)*time.Hour)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		resp := struct {
			Token string `json:"token"`
		}{
			Token: access_token,
		}
		dat, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(dat)
	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		refresh_token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		err = apiConfig.db.RevokeRefreshToken(r.Context(), refresh_token)
		if err != nil {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(204)
	})
	server := http.Server{Handler: mux}
	server.Addr = ":8080"
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
