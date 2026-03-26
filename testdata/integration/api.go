package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func HandleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users := []string{"alice", "bob"}
		json.NewEncoder(w).Encode(Response{Status: "ok", Data: users})
	case http.MethodPost:
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Status: "error", Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(Response{Status: "created", Data: body})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
