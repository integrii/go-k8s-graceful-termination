package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
)

// ShuttingDown globally calls out if our app is shutting down or not
var ShuttingDown bool

func main() {

	// watch for shutdown signals
	go trapShutdownSignal()

	// setup our main index handler
	http.HandleFunc("/", indexHandler)
	// setup a liveness handler
	http.HandleFunc("/alive", livenessHandler)
	// setup a readiness handler
	http.HandleFunc("/ready", readinessHandler)

	// start our web service
	err := http.ListenAndServe(":8080", http.DefaultServeMux)
	if err != nil {
		log.Fatalln("error when running web server:", err)
	}
}

func trapShutdownSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	ShuttingDown = true
}

// indexHandler handles requests that don't match any other handler path
func indexHandler(res http.ResponseWriter, req *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to get my hostname:", err)
		return
	}
	res.Write([]byte("Hello! I am here! - " + hostname))
}

// readinessHandler handles requests that check the applications' readiness.
// This is mainly used by the readiness probes.  Failures on this endpoint
// will result in this pod being removed from the service endpoint list
// where new traffic is sent.
func readinessHandler(res http.ResponseWriter, req *http.Request) {
	if ShuttingDown {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// livenessHandler handles requests that check the applications' liveness.
// This is mainly used by the liveness probes.  Failures on this endpoint
// will result in the app being restarted by Kubernetes.
func livenessHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}
