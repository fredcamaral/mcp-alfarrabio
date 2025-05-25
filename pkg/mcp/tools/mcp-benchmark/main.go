package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

const version = "1.0.0"

type Benchmark struct {
	serverPath  string
	concurrency int
	duration    time.Duration
	requestType string
	verbose     bool

	// Metrics
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	latencies       []time.Duration
	mu              sync.Mutex
}

type Worker struct {
	id     int
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	dec    *json.Decoder
	enc    *json.Encoder
}

func main() {
	var (
		serverPath  = flag.String("server", "", "Path to MCP server executable")
		concurrency = flag.Int("concurrency", 1, "Number of concurrent connections")
		duration    = flag.Duration("duration", 60*time.Second, "Benchmark duration")
		requestType = flag.String("type", "tools/list", "Request type to benchmark")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
		showVersion = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("MCP Benchmark v%s\n", version)
		os.Exit(0)
	}

	if *serverPath == "" {
		if flag.NArg() > 0 {
			*serverPath = flag.Arg(0)
		} else {
			fmt.Fprintf(os.Stderr, "Error: server path required\n")
			flag.Usage()
			os.Exit(1)
		}
	}

	benchmark := &Benchmark{
		serverPath:  *serverPath,
		concurrency: *concurrency,
		duration:    *duration,
		requestType: *requestType,
		verbose:     *verbose,
		latencies:   make([]time.Duration, 0, 10000),
	}

	if err := benchmark.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Benchmark error: %v\n", err)
		os.Exit(1)
	}
}

func (b *Benchmark) Run() error {
	fmt.Printf("Starting MCP Benchmark\n")
	fmt.Printf("======================\n")
	fmt.Printf("Server: %s\n", b.serverPath)
	fmt.Printf("Concurrency: %d\n", b.concurrency)
	fmt.Printf("Duration: %s\n", b.duration)
	fmt.Printf("Request Type: %s\n\n", b.requestType)

	// Create workers
	workers := make([]*Worker, b.concurrency)
	for i := 0; i < b.concurrency; i++ {
		worker, err := NewWorker(i, b.serverPath)
		if err != nil {
			return fmt.Errorf("create worker %d: %w", i, err)
		}
		defer worker.Close()

		// Initialize worker
		if err := worker.Initialize(); err != nil {
			return fmt.Errorf("initialize worker %d: %w", i, err)
		}

		workers[i] = worker
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start benchmark
	ctx, cancel := context.WithTimeout(context.Background(), b.duration)
	defer cancel()

	start := time.Now()
	var wg sync.WaitGroup

	// Start workers
	for _, worker := range workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			b.runWorker(ctx, w)
		}(worker)
	}

	// Progress reporter
	go b.reportProgress(ctx)

	// Wait for completion or interrupt
	select {
	case <-ctx.Done():
		fmt.Printf("\nBenchmark completed\n")
	case <-sigChan:
		fmt.Printf("\nBenchmark interrupted\n")
		cancel()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Print results
	b.printResults(elapsed)

	return nil
}

func NewWorker(id int, serverPath string) (*Worker, error) {
	cmd := exec.Command(serverPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	return &Worker{
		id:     id,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		dec:    json.NewDecoder(stdout),
		enc:    json.NewEncoder(stdin),
	}, nil
}

func (w *Worker) Initialize() error {
	// Send initialize request
	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "1.0",
			"capabilities": {},
			"clientInfo": {
				"name": "mcp-benchmark",
				"version": "` + version + `"
			}
		}`),
	}

	if err := w.enc.Encode(req); err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}

	// Read response
	var resp protocol.JSONRPCResponse
	if err := w.dec.Decode(&resp); err != nil {
		return fmt.Errorf("read initialize response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	notif := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := w.enc.Encode(notif); err != nil {
		return fmt.Errorf("send initialized: %w", err)
	}

	return nil
}

func (b *Benchmark) runWorker(ctx context.Context, w *Worker) {
	requestID := int64(w.id * 1000000) // Ensure unique IDs across workers

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		requestID++
		start := time.Now()

		// Send request
		req := protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(fmt.Sprintf(`%d`, requestID)),
			Method:  b.requestType,
		}

		if err := w.enc.Encode(req); err != nil {
			atomic.AddInt64(&b.failedRequests, 1)
			if b.verbose {
				fmt.Printf("Worker %d: send error: %v\n", w.id, err)
			}
			continue
		}

		// Read response
		var resp protocol.JSONRPCResponse
		if err := w.dec.Decode(&resp); err != nil {
			atomic.AddInt64(&b.failedRequests, 1)
			if b.verbose {
				fmt.Printf("Worker %d: read error: %v\n", w.id, err)
			}
			continue
		}

		latency := time.Since(start)
		atomic.AddInt64(&b.totalRequests, 1)

		if resp.Error != nil {
			atomic.AddInt64(&b.failedRequests, 1)
			if b.verbose {
				fmt.Printf("Worker %d: response error: %s\n", w.id, resp.Error.Message)
			}
		} else {
			atomic.AddInt64(&b.successRequests, 1)
			b.recordLatency(latency)
		}
	}
}

func (b *Benchmark) recordLatency(latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.latencies = append(b.latencies, latency)
}

func (b *Benchmark) reportProgress(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&b.totalRequests)
			success := atomic.LoadInt64(&b.successRequests)
			failed := atomic.LoadInt64(&b.failedRequests)
			fmt.Printf("Progress: %d requests (%d success, %d failed)\n", total, success, failed)
		}
	}
}

func (b *Benchmark) printResults(elapsed time.Duration) {
	total := atomic.LoadInt64(&b.totalRequests)
	success := atomic.LoadInt64(&b.successRequests)
	failed := atomic.LoadInt64(&b.failedRequests)

	fmt.Printf("\nBenchmark Results\n")
	fmt.Printf("=================\n\n")

	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful: %d (%.2f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Failed: %d (%.2f%%)\n", failed, float64(failed)/float64(total)*100)
	fmt.Printf("Duration: %s\n", elapsed)
	fmt.Printf("\n")

	if total > 0 {
		rps := float64(total) / elapsed.Seconds()
		fmt.Printf("Requests/sec: %.2f\n", rps)
	}

	// Latency statistics
	b.mu.Lock()
	latencies := b.latencies
	b.mu.Unlock()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		fmt.Printf("\nLatency Statistics\n")
		fmt.Printf("------------------\n")
		fmt.Printf("Min: %v\n", latencies[0])
		fmt.Printf("Max: %v\n", latencies[len(latencies)-1])
		fmt.Printf("Mean: %v\n", mean(latencies))
		fmt.Printf("Median: %v\n", percentile(latencies, 50))
		fmt.Printf("P90: %v\n", percentile(latencies, 90))
		fmt.Printf("P95: %v\n", percentile(latencies, 95))
		fmt.Printf("P99: %v\n", percentile(latencies, 99))
		fmt.Printf("StdDev: %v\n", stdDev(latencies))
	}

	// Throughput chart
	fmt.Printf("\nThroughput Chart\n")
	fmt.Printf("----------------\n")
	printThroughputChart(total, elapsed)
}

func (w *Worker) Close() error {
	if w.stdin != nil {
		w.stdin.Close()
	}
	if w.cmd != nil {
		w.cmd.Process.Kill()
		w.cmd.Wait()
	}
	return nil
}

func mean(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}

func percentile(latencies []time.Duration, p float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	idx := int(float64(len(latencies)-1) * p / 100)
	return latencies[idx]
}

func stdDev(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	m := mean(latencies)
	var sum float64

	for _, l := range latencies {
		diff := float64(l - m)
		sum += diff * diff
	}

	variance := sum / float64(len(latencies))
	return time.Duration(math.Sqrt(variance))
}

func printThroughputChart(total int64, elapsed time.Duration) {
	rps := float64(total) / elapsed.Seconds()
	maxWidth := 50
	normalizedRPS := int(rps / 100) // Scale down for display

	if normalizedRPS > maxWidth {
		normalizedRPS = maxWidth
	}

	fmt.Printf("RPS: ")
	for i := 0; i < normalizedRPS; i++ {
		fmt.Print("█")
	}
	fmt.Printf(" %.2f req/s\n", rps)
}