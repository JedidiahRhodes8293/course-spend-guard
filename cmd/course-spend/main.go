package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"course-spend-guard/coursebudget"
)

func main() {
	api, err := coursebudget.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/deliver-lesson", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var input coursebudget.LessonRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := coursebudget.DeliverLesson(r.Context(), api, input)
		if err != nil {
			var apiErr *coursebudget.APIError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				http.Error(w, apiErr.Message, apiErr.Status)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	server := &http.Server{Addr: ":8080", ReadHeaderTimeout: 5 * time.Second}
	log.Printf("course spend guard listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
