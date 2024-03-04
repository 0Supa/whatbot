package httpServer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	discordClient "github.com/0supa/degen/client/discord"
	"github.com/0supa/degen/client/pwd"
	"github.com/0supa/degen/client/sql"
	"github.com/0supa/degen/client/store"
	"github.com/diamondburned/arikawa/v3/api"
	regexp "github.com/wasilibs/go-re2"
)

type obj = map[string]interface{}

var webAddr = "localhost:9987"

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Key      string `json:"key"`
}

func resError(w http.ResponseWriter, message string, statusCode int) {
	m := obj{
		"message": message,
		"error":   statusCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(m)
}

var mu = sync.Mutex{}

func init() {
	fs := http.FileServer(http.Dir("./web/build"))
	http.Handle("/mc/register/", http.StripPrefix("/mc/register/", fs))

	http.HandleFunc("/mc/api/user", func(w http.ResponseWriter, r *http.Request) {
		u := store.RegisterKeys[r.FormValue("key")]
		if u == nil || time.Now().After(u.Expiry) {
			resError(w, "key not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj{
			"name":   u.User.Username,
			"avatar": u.User.AvatarURL(),
		})
	})

	http.HandleFunc("/mc/api/whitelist", func(w http.ResponseWriter, r *http.Request) {
		_, err := sql.GetPlayer(r.FormValue("name"), "")

		w.Header().Set("Content-Type", "application/json")

		if err == nil {
			json.NewEncoder(w).Encode(obj{
				"found": true,
			})
			return
		}
		if err != sql.ErrNil {
			resError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(obj{
			"found": false,
		})
	})

	http.HandleFunc("/mc/api/register", func(w http.ResponseWriter, r *http.Request) {
		var p registerRequest

		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			resError(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		discordReq := store.RegisterKeys[p.Key]
		if discordReq == nil {
			resError(w, "unknown request key", http.StatusBadRequest)
			return
		}

		if time.Now().After(discordReq.Expiry) {
			resError(w, "your request key has expired", http.StatusBadRequest)
			return
		}

		if validName := regexp.MustCompile("^[a-zA-Z0-9_]{3,16}$").MatchString(p.Username); !validName {
			resError(w, "invalid username provided", http.StatusBadRequest)
			return
		}

		if len(p.Password) > 32 || len(p.Password) < 8 {
			resError(w, "password does not meet the requirements", http.StatusBadRequest)
			return
		}

		_, err := sql.GetPlayer(p.Username, "")

		if err == nil {
			resError(w, "player already whitelisted", http.StatusBadRequest)
			return
		}

		if err != sql.ErrNil {
			resError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		delete(store.RegisterKeys, p.Key)

		salt := pwd.GenerateSalt(32)
		saltHex := hex.EncodeToString(salt)

		hash := pwd.Hash(p.Password, salt)

		_, err = sql.DB.Exec("INSERT INTO passky_players (uuid, algo, hash, salt, ip, date) VALUES (?, ?, ?, ?, ?, ?)", p.Username, "SHA-512", hash, saltHex, r.Header.Get("cf-connecting-ip"), fmt.Sprint(time.Now().UnixMilli()))
		if err != nil {
			resError(w, err.Error(), http.StatusInternalServerError)
		}

		_, err = sql.DB.Exec("INSERT INTO noble_whitelist (Name, UUID, Discord, Whitelisted) VALUES (?, ?, ?, ?)", p.Username, nil, discordReq.User.ID, 1)
		if err != nil {
			resError(w, err.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj{
			"success": true,
		})

		go func() {
			err := discordClient.Handler.AddRole(1200915706661843074, discordReq.User.ID, 1206804152513208340, api.AddRoleData{})
			if err != nil {
				log.Println("failed adding role:", err)
			}
		}()
	})

	go func() {
		log.Println("WWW running on " + webAddr)
		log.Fatal(http.ListenAndServe(webAddr, nil))
	}()
}
