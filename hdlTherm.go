package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

//
// Get status handler

func (s *server) getStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Checking API key
		// TODO: might be better to replace with a constant-time implementation
		// see https://pkg.go.dev/crypto/subtle
		keys, ok := r.URL.Query()["key"]
		if !ok || len(keys[0]) < 1 {
			s.logger.Println("Missing API key parameter")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if keys[0] != s.thermKey {
			s.logger.Println("Invalid key")
			http.Error(w, "Invalid key, access denied", http.StatusForbidden)
			return
		}

		// Reassuring watchdog that things seem to be working fine
		s.watchdog.Check()

		w.Header().Set("Content-type", "application/json; charset=UTF-8")
		err := json.NewEncoder(w).Encode(s.therm.GetState())
		if err != nil {
			s.logger.Println("Error marshalling response status to JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

//
// Post relay state handler

func (s *server) postSet() http.HandlerFunc {
	type request struct {
		State thermState `json:"state"`
		Key   string     `json:"key"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsing and checking JSON request
		var req request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.logger.Println("Error parsing input: ", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Checking API key
		// TODO: might be better to replace with a constant-time implementation
		// see https://pkg.go.dev/crypto/subtle
		if req.Key != s.thermKey {
			s.logger.Println("Invalid key")
			http.Error(w, "Invalid key, access denied", http.StatusForbidden)
			return
		}

		// Reassuring watchdog that things seem to be working fine
		s.watchdog.Check()

		// Setting relay state
		if err := s.therm.SetState(req.State); err != nil {
			s.logger.Println("Error setting state to", req.State, ":", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Writing response
		s.logger.Println("State set to", req.State)
		fmt.Fprint(w, "OK")
	}
}
