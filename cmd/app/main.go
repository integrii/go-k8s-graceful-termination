package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	log.Println("starting web service on :8080")
	err := http.ListenAndServe(":8080", http.DefaultServeMux)
	if err != nil {
		log.Fatalln("error when running web server:", err)
	}
	os.Exit(0)
}

func trapShutdownSignal() {
	log.Println("watching for termination signals")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	// when we get a signal, flip the global ShuttingDown flag
	sig := <-sigChan
	log.Println("got signal:", sig)
	ShuttingDown = true

	// wait for the liveness checks to fail and kubernetes to reconfigure
	log.Println("graceful shutdown has begun")
	time.Sleep(time.Second * 20) // periodSeconds * failureThreshold + 10s
	log.Println("exiting clean due to shutdown signal")
	os.Exit(0)
}

// indexHandler handles requests that don't match any other handler path
func indexHandler(res http.ResponseWriter, req *http.Request) {
	log.Println("handing normal web request from", req.Host)
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
		log.Println("failing readiness request from", req.Host)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("handing readiness request from", req.Host)
	res.WriteHeader(http.StatusOK)
}

// livenessHandler handles requests that check the applications' liveness.
// This is mainly used by the liveness probes.  Failures on this endpoint
// will result in the app being restarted by Kubernetes.
func livenessHandler(res http.ResponseWriter, req *http.Request) {
	log.Println("handing liveness request from", req.Host)
	res.WriteHeader(http.StatusOK)
}
