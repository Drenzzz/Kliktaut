// Load test for the redirect endpoint (§2.2 DoD: p50 < 50ms on cache hit).
// Reports p50/p95/p99 and throughput. Two phases:
//   miss  - first round, fills the cache (excluded from the DoD claim)
//   hit   - measured rounds, all served from cache
package main

import (
	"flag"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type result struct {
	durations []float64 // ms
	errors    int64
}

func runPhase(client *http.Client, url string, total, concurrency int) result {
	var wg sync.WaitGroup
	var errs atomic.Int64
	dur := make([]float64, 0, total)
	var mu sync.Mutex

	per := total / concurrency
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]float64, 0, per)
			for i := 0; i < per; i++ {
				start := time.Now()
				resp, err := client.Get(url)
				elapsed := float64(time.Since(start).Microseconds()) / 1000.0
				if err != nil || resp.StatusCode != http.StatusFound {
					errs.Add(1)
				}
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
				local = append(local, elapsed)
			}
			mu.Lock()
			dur = append(dur, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Float64s(dur)
	return result{durations: dur, errors: errs.Load()}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p / 100 * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func report(name string, r result, wall time.Duration) {
	fmt.Printf("\n%s:\n", name)
	fmt.Printf("  requests : %d (%d errors)\n", len(r.durations), r.errors)
	fmt.Printf("  p50      : %.2f ms\n", percentile(r.durations, 50))
	fmt.Printf("  p95      : %.2f ms\n", percentile(r.durations, 95))
	fmt.Printf("  p99      : %.2f ms\n", percentile(r.durations, 99))
	fmt.Printf("  throughput: %.0f req/s\n", float64(len(r.durations))/wall.Seconds())
}

func main() {
	url := flag.String("url", "http://localhost:3000", "base URL")
	code := flag.String("code", "kliktaut-docs", "short code to hit")
	total := flag.Int("total", 5000, "requests per phase")
	concurrency := flag.Int("concurrency", 50, "concurrent workers")
	flag.Parse()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
		},
		// 302 is the expected answer; do not follow it
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	target := *url + "/" + *code

	fmt.Printf("load test: %s | total=%d | concurrency=%d\n", target, *total, *concurrency)

	// warm-up + miss phase (fills the cache, not part of the DoD claim)
	miss := runPhase(client, target, *total/5, *concurrency)
	report("Phase 1: cache miss (cold, fills cache)", miss, time.Duration(0))

	time.Sleep(200 * time.Millisecond) // let async click writes drain

	start := time.Now()
	hit := runPhase(client, target, *total, *concurrency)
	wall := time.Since(start)
	report("Phase 2: cache hit (DoD measurement)", hit, wall)

	p50 := percentile(hit.durations, 50)
	fmt.Printf("\nDoD §2.2 (p50 < 50ms on cache hit): ")
	if p50 < 50 && hit.errors == 0 {
		fmt.Printf("PASS (%.2f ms)\n", p50)
	} else {
		fmt.Printf("FAIL (p50=%.2f ms, errors=%d). See SC-014\n", p50, hit.errors)
	}
}
