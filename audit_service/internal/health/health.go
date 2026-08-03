package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Checker func(ctx context.Context) error

type checkResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type response struct {
	Status string                 `json:"status"`
	Checks map[string]checkResult `json:"checks"`
}

func Handler(timeout time.Duration, checks map[string]Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		results := make(map[string]checkResult, len(checks))
		var mu sync.Mutex
		var wg sync.WaitGroup
		for name, check := range checks {
			wg.Add(1)
			go func(name string, check Checker) {
				defer wg.Done()
				err := check(ctx)
				mu.Lock()
				if err != nil {
					results[name] = checkResult{OK: false, Error: err.Error()}
				} else {
					results[name] = checkResult{OK: true}
				}
				mu.Unlock()
			}(name, check)
		}
		wg.Wait()

		status := "ok"
		code := http.StatusOK
		for _, c := range results {
			if !c.OK {
				status = "degraded"
				code = http.StatusServiceUnavailable
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(response{Status: status, Checks: results})
	}
}

func DB(db *sql.DB) Checker {
	return func(ctx context.Context) error {
		return db.PingContext(ctx)
	}
}

func Kafka(brokers []string) Checker {
	return func(ctx context.Context) error {
		if len(brokers) == 0 {
			return nil
		}
		dialer := &kafka.Dialer{Timeout: 3 * time.Second, DualStack: true}
		var lastErr error
		for _, b := range brokers {
			conn, err := dialer.DialContext(ctx, "tcp", b)
			if err != nil {
				lastErr = err
				continue
			}
			_ = conn.Close()
			return nil
		}
		if lastErr == nil {
			lastErr = &net.OpError{Op: "dial", Err: errAllBrokersUnreachable}
		}
		return lastErr
	}
}

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errAllBrokersUnreachable sentinelErr = "all brokers unreachable"
