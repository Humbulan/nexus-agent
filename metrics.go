package main

import (
"flag"
"fmt"
"net/http"
"os"
"strings"
	"sync"
)

var metricsAddr string

func init() {
flag.StringVar(&metricsAddr, "metrics-addr", "", "if set, serve Prometheus metrics on this address (e.g. :9090)")
}

type metricsState struct {
mu      sync.Mutex
up      map[string]bool
latency map[string]float64
lastRun float64
}

var metrics = &metricsState{
up:      make(map[string]bool),
latency: make(map[string]float64),
}

func recordCheck(name string, isUp bool, ms float64) {
metrics.mu.Lock()
defer metrics.mu.Unlock()
metrics.up[name] = isUp
metrics.latency[name] = ms
}

func recordRun(ms float64) {
metrics.mu.Lock()
defer metrics.mu.Unlock()
metrics.lastRun = ms
}

func startMetricsServer(addr string) {
mux := http.NewServeMux()
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
metrics.mu.Lock()
defer metrics.mu.Unlock()
w.Header().Set("Content-Type", "text/plain; version=0.0.4")
fmt.Fprintln(w, "# HELP nexus_service_up 1 if service is reachable, 0 if not")
fmt.Fprintln(w, "# TYPE nexus_service_up gauge")
for name, up := range metrics.up {
v := 0
if up {
v = 1
}
fmt.Fprintf(w, "nexus_service_up{service=%q} %d\n", name, v)
}
fmt.Fprintln(w, "# HELP nexus_service_latency_ms Last check latency in milliseconds")
fmt.Fprintln(w, "# TYPE nexus_service_latency_ms gauge")
for name, ms := range metrics.latency {
fmt.Fprintf(w, "nexus_service_latency_ms{service=%q} %f\n", name, ms)
}
fmt.Fprintln(w, "# HELP nexus_last_run_ms Duration of the last full run")
fmt.Fprintln(w, "# TYPE nexus_last_run_ms gauge")
fmt.Fprintf(w, "nexus_last_run_ms %f\n", metrics.lastRun)
})
mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
fmt.Fprintln(w, "ok")
})
go func() {
if err := http.ListenAndServe(addr, mux); err != nil {
fmt.Fprintln(os.Stderr, "metrics server:", err)
}
}()
fmt.Fprintln(os.Stderr, "metrics: listening on", addr+"/metrics")
}

func pushToPushgateway() {
url := os.Getenv("NEXUS_PUSHGATEWAY")
if url == "" {
return
}
metrics.mu.Lock()
var buf strings.Builder
buf.WriteString("# TYPE nexus_service_up gauge\n")
for name, up := range metrics.up {
v := 0
if up {
v = 1
}
fmt.Fprintf(&buf, "nexus_service_up{service=%q} %d\n", name, v)
}
buf.WriteString("# TYPE nexus_service_latency_ms gauge\n")
for name, ms := range metrics.latency {
fmt.Fprintf(&buf, "nexus_service_latency_ms{service=%q} %f\n", name, ms)
}
buf.WriteString("# TYPE nexus_last_run_ms gauge\n")
fmt.Fprintf(&buf, "nexus_last_run_ms %f\n", metrics.lastRun)
metrics.mu.Unlock()

endpoint := strings.TrimRight(url, "/") + "/metrics/job/nexus-agent"
resp, err := http.Post(endpoint, "text/plain", strings.NewReader(buf.String()))
if err != nil {
fmt.Fprintln(os.Stderr, "pushgateway push failed:", err)
return
}
defer resp.Body.Close()
if resp.StatusCode >= 300 {
fmt.Fprintln(os.Stderr, "pushgateway returned status", resp.StatusCode)
return
}
fmt.Fprintln(os.Stderr, "metrics pushed to pushgateway")
}
