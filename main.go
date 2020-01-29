package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var scanningService ScanningService = ScanningService{}

// Google cloud run friendly listener string generator
func getListenerString() string {
	host, hb := os.LookupEnv("HOST")
	port, pb := os.LookupEnv("PORT")

	if !pb {
		port = "8000"
	}

	if !hb {
		host = "0.0.0.0"
	}

	return fmt.Sprintf("%s:%s", host, port)
}

func initLogger() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stderr)
	// log.SetLevel(log.InfoLevel)
	log.SetLevel(log.DebugLevel)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadFile("versions.json")
	if err == nil {
		w.Write(content)
	} else {
		w.Write([]byte("Failed to load versions.json"))
	}
}

func default404Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Not Found"))
}

func scanSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	scanRequest := ScanRequest{}

	w.Header().Add("Content-Type", "application/json")
	if err := decoder.Decode(&scanRequest); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to decode request params"})
	} else {
		// scanReport, _ := scanningService.ScanImage(scanRequest)
		// json.NewEncoder(w).Encode(scanReport)
		scanID := scanningService.AsyncScanImage(scanRequest)
		json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
	}
}

func scanStatusHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	json.NewEncoder(w).Encode(map[string]string{"status": scanningService.GetScanStatus(params["scan_id"])})
}

func corsMiddleware(r *mux.Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			next.ServeHTTP(w, req)
		})
	}
}

func main() {
	initLogger()
	scanningService.Init()

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/version", versionHandler).Methods("GET")
	r.HandleFunc("/scans/{scan_id}/status", scanStatusHandler).Methods("GET")
	r.HandleFunc("/scans/{scan_id}", default404Handler).Methods("GET")
	r.HandleFunc("/scans", scanSubmissionHandler).Methods("POST")
	r.Use(corsMiddleware(r))

	loggingRouter := handlers.LoggingHandler(os.Stdout, r)

	log.Infof("Starting HTTP server on %s", getListenerString())
	http.ListenAndServe(getListenerString(), loggingRouter)
}
