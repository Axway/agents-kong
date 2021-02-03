package server

import (
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Axway/agent-sdk/pkg/util/log"
)

func NewTraceServer() {
	http.HandleFunc("/logs", logHandler)
	log.Info("Starting Kong Traceability Agent on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func logHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}

	// authenticate the request

	// Is it only json?
	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(400)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	log.Info(string(body))

	w.WriteHeader(200)
	fmt.Fprint(w, "Kong Traceability Agent")
}
