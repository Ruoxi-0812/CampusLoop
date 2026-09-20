package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// Opt-in baseline: real RPCs over loopback TCP, with client and server in one process.
// This deliberately excludes frontend, Kubernetes, databases and external networks.
func TestPerformanceBaseline(t *testing.T) {
	output := os.Getenv("CATALOG_PERF_OUTPUT")
	if output == "" {
		t.Skip("set CATALOG_PERF_OUTPUT to save a baseline")
	}
	t.Setenv("ALLOYDB_CLUSTER_NAME", "")
	reloadCatalog.Store(false)
	extraLatency = 0
	svc := &productCatalog{}
	if err := loadCatalog(&svc.catalog); err != nil {
		t.Fatal(err)
	}
	if raw := os.Getenv("CATALOG_PERF_SIZE"); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 {
			t.Fatal("invalid CATALOG_PERF_SIZE")
		}
		original := svc.catalog.Products
		synthetic := make([]*pb.Product, size)
		for i := range synthetic {
			synthetic[i] = proto.Clone(original[i%len(original)]).(*pb.Product)
			synthetic[i].Id = fmt.Sprintf("synthetic-%08d", i)
		}
		svc.catalog.Products = synthetic
	}
	products := svc.catalog.Products
	if len(products) == 0 {
		t.Fatal("empty catalog")
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterProductCatalogServiceServer(server, svc)
	go server.Serve(lis)
	defer server.Stop()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewProductCatalogServiceClient(conn)
	type result struct {
		Method       string  `json:"method"`
		Concurrency  int     `json:"concurrency"`
		Repeat       int     `json:"repeat"`
		Requests     int     `json:"requests"`
		Errors       int     `json:"errors"`
		Seconds      float64 `json:"seconds"`
		RPS          float64 `json:"rps"`
		ErrorPercent float64 `json:"error_percent"`
		P50          float64 `json:"p50_ms"`
		P95          float64 `json:"p95_ms"`
		P99          float64 `json:"p99_ms"`
	}
	var results []result
	concurrencies := []int{1, 20}
	methods := []string{"GetProduct", "ListProducts"}
	if os.Getenv("CATALOG_PERF_LOOKUP_ONLY") == "1" {
		concurrencies = []int{20}
		methods = []string{"GetProduct"}
	}
	for repeat := 1; repeat <= 3; repeat++ {
		for _, concurrency := range concurrencies {
			for _, method := range methods {
				call := func(i int) error {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					if method == "GetProduct" {
						want := products[i%len(products)]
						got, err := client.GetProduct(ctx, &pb.GetProductRequest{Id: want.Id})
						if err != nil {
							return err
						}
						if !proto.Equal(got, want) {
							return fmt.Errorf("incorrect product")
						}
					} else {
						got, err := client.ListProducts(ctx, &pb.Empty{})
						if err != nil {
							return err
						}
						if !proto.Equal(got, &svc.catalog) {
							return fmt.Errorf("incorrect catalog")
						}
					}
					return nil
				}
				for i := 0; i < 100; i++ {
					if err := call(i); err != nil {
						t.Fatal(err)
					}
				}
				var wg sync.WaitGroup
				var mu sync.Mutex
				var samples []float64
				errors := 0
				start := time.Now()
				deadline := start.Add(5 * time.Second)
				for worker := 0; worker < concurrency; worker++ {
					wg.Add(1)
					go func(offset int) {
						defer wg.Done()
						local := make([]float64, 0, 10000)
						failed := 0
						for i := offset; time.Now().Before(deadline); i += concurrency {
							began := time.Now()
							err := call(i)
							local = append(local, float64(time.Since(began).Nanoseconds())/1e6)
							if err != nil {
								failed++
							}
						}
						mu.Lock()
						samples = append(samples, local...)
						errors += failed
						mu.Unlock()
					}(worker)
				}
				wg.Wait()
				elapsed := time.Since(start).Seconds()
				sort.Float64s(samples)
				percentile := func(p float64) float64 { return samples[int(math.Ceil(p*float64(len(samples))))-1] }
				r := result{method, concurrency, repeat, len(samples), errors, elapsed, float64(len(samples)) / elapsed, 100 * float64(errors) / float64(len(samples)), percentile(.50), percentile(.95), percentile(.99)}
				results = append(results, r)
				t.Logf("%s concurrency=%d repeat=%d rps=%.0f p95=%.3fms errors=%d", method, concurrency, repeat, r.RPS, r.P95, errors)
			}
		}
	}
	report := map[string]interface{}{"synthetic": os.Getenv("CATALOG_PERF_SIZE") != "", "variant": os.Getenv("CATALOG_PERF_VARIANT"), "timestamp": time.Now().UTC().Format(time.RFC3339), "go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "catalog_size": len(products), "transport": "loopback TCP; one shared HTTP/2 connection; same-process client/server; no tracing interceptors", "warmup_requests_per_case": 100, "seconds_per_case": 5, "results": results}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Errors > 0 {
			t.Errorf("%s had %d errors", r.Method, r.Errors)
		}
	}
}
