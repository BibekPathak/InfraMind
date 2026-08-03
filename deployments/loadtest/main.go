package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type telemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Current     float64 `json:"current"`
	Voltage     float64 `json:"voltage"`
	Humidity    float64 `json:"humidity"`
	Scenario    string  `json:"scenario"`
}

type result struct {
	ok       int64
	fail     int64
	latencies []float64
	mu       sync.Mutex
}

func (r *result) record(lat time.Duration, success bool) {
	if success {
		atomic.AddInt64(&r.ok, 1)
	} else {
		atomic.AddInt64(&r.fail, 1)
	}
	r.mu.Lock()
	r.latencies = append(r.latencies, lat.Seconds())
	r.mu.Unlock()
}

func main() {
	mode := flag.String("mode", "mqtt", "test mode: mqtt | api")
	broker := flag.String("broker", "mqtt://localhost:1883", "MQTT broker URL")
	topic := flag.String("topic", "telemetry/tx-001/data", "MQTT publish topic")
	url := flag.String("url", "http://localhost:8080/api/v1", "backend API base URL")
	rate := flag.Int("rate", 50, "requests per second")
	seconds := flag.Int("seconds", 10, "test duration (seconds)")
	devices := flag.Int("devices", 1, "number of simulated devices (mqtt mode)")
	flag.Parse()

	log.Printf("load test mode=%s rate=%d/s duration=%ds", *mode, *rate, *seconds)

	var res result
	total := *rate * *seconds

	switch *mode {
	case "mqtt":
		runMQTT(*broker, *topic, *rate, *seconds, *devices, &res)
	case "api":
		runAPI(*url, *rate, *seconds, &res)
	default:
		log.Fatalf("unknown mode %q", *mode)
	}

	// Report
	res.mu.Lock()
	latencies := res.latencies
	res.mu.Unlock()

	var sum, p50, p95, p99, max float64
	if len(latencies) > 0 {
		sortFloats(latencies)
		sum = sumFloats(latencies)
		p50 = latencies[int(0.50*float64(len(latencies)))]
		p95 = latencies[int(0.95*float64(len(latencies)))]
		p99 = latencies[int(0.99*float64(len(latencies)))]
		max = latencies[len(latencies)-1]
	}

	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Total:            %d\n", total)
	fmt.Printf("Successful:       %d\n", res.ok)
	fmt.Printf("Errors:           %d\n", res.fail)
	if len(latencies) > 0 {
		fmt.Printf("Avg latency:      %.3fs\n", sum/float64(len(latencies)))
		fmt.Printf("p50:              %.3fs\n", p50)
		fmt.Printf("p95:              %.3fs\n", p95)
		fmt.Printf("p99:              %.3fs\n", p99)
		fmt.Printf("Max:              %.3fs\n", max)
		fmt.Printf("Throughput:       %.1f req/s\n", float64(res.ok)/float64(*seconds))
	} else {
		fmt.Println("No requests completed")
	}
}

func runMQTT(broker, topic string, rate, seconds, devices int, res *result) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("loadtest-%d", time.Now().UnixNano()))
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		log.Fatalf("mqtt connect: %v", tok.Error())
	}
	defer client.Disconnect(250)

	var seq int64
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()
	deadline := time.After(time.Duration(seconds) * time.Second)

	for {
		select {
		case <-deadline:
			return
		case <-ticker.C:
			n := atomic.AddInt64(&seq, 1)
			deviceID := fmt.Sprintf("load-%d", (n%int64(devices))+1)
			payload := telemetryPayload{
				DeviceID:    deviceID,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				Temperature: 60 + float64(n%30),
				Current:     90 + float64(n%40),
				Voltage:     11400,
				Humidity:    45,
				Scenario:    "loadtest",
			}
			data, _ := json.Marshal(payload)

			start := time.Now()
			tok := client.Publish(topic, 1, false, data)
			done := make(chan struct{})
			go func() {
				tok.Wait()
				close(done)
			}()
			select {
			case <-done:
				res.record(time.Since(start), tok.Error() == nil)
			case <-time.After(5 * time.Second):
				res.record(time.Since(start), false)
			}
		}
	}
}

func runAPI(baseURL string, rate, seconds int, res *result) {
	client := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()
	deadline := time.After(time.Duration(seconds) * time.Second)

	for {
		select {
		case <-deadline:
			return
		case <-ticker.C:
			start := time.Now()
			url := baseURL + "/telemetry/live?device_id=tx-001"
			resp, err := client.Get(url)
			success := err == nil && resp.StatusCode == http.StatusOK
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			res.record(time.Since(start), success)
		}
	}
}

func sortFloats(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sumFloats(s []float64) float64 {
	var sum float64
	for _, v := range s {
		sum += v
	}
	return sum
}
