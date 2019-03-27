package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ocmmetrics "github.com/slok/go-http-metrics/metrics/opencensus"
	"github.com/slok/go-http-metrics/middleware"
	ocprometheus "go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

const (
	srvAddr     = ":8080"
	metricsAddr = ":8081"
)

// this example will show the simplest way of enabling the middleware
// using go std handlers without frameworks with the Prometheues recorder.
// it uses the prometheus default registry, the middleware default options
// and doesn't set the handler ID, so the middleware will set the handler id
// (handler label) to the request URL, WARNING: this creates high cardinality because
// `/profile/123` and `/profile/567` are different handlers. If you want to be safer
// you will need to tell the middleware what is the handler id.
func main() {
	// Create OpenCensus with Prometheus.
	ocexporter, err := ocprometheus.NewExporter(ocprometheus.Options{})
	if err != nil {
		log.Panicf("error creating OpenCensus exporter: %s", err)
	}
	view.RegisterExporter(ocexporter)
	rec, err := ocmmetrics.NewRecorder(ocmmetrics.Config{})
	if err != nil {
		log.Panicf("error creating OpenCensus metrics recorder: %s", err)
	}

	// Create our middleware.
	mdlw := middleware.New(middleware.Config{
		Recorder: rec,
	})

	// Create our server.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/test1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })
	mux.HandleFunc("/test1/test2", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) })
	mux.HandleFunc("/test1/test4", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNonAuthoritativeInfo) })
	mux.HandleFunc("/test2", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/test3", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusResetContent) })

	// Wrap our main handler, we pass empty handler ID so the middleware inferes
	// the handler label from the URL.
	h := mdlw.Handler("", mux)

	// Serve our handler.
	go func() {
		log.Printf("server listening at %s", srvAddr)
		if err := http.ListenAndServe(srvAddr, h); err != nil {
			log.Panicf("error while serving: %s", err)
		}
	}()

	// Serve our metrics.
	go func() {
		log.Printf("metrics listening at %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, ocexporter); err != nil {
			log.Panicf("error while serving metrics: %s", err)
		}
	}()

	// Wait until some signal is captured.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
}
