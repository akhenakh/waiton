package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_HTTPTest_Working(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	slowHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
	}

	ts := httptest.NewServer(http.HandlerFunc(slowHandler))
	defer ts.Close()

	if err := httpTest(ctx, http.DefaultClient, ts.URL, 1); err != nil {
		t.Fatalf("should be a working request got err %v", err)
	}
}

func Test_HTTPTest_No200(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	badHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	ts := httptest.NewServer(http.HandlerFunc(badHandler))
	defer ts.Close()

	client := &http.Client{Timeout: 1 * time.Second}

	if err := httpTest(ctx, client, ts.URL, 4); err == nil {
		t.Fatalf("should be a failing request got no err")
	}
}

func Test_HTTPTest_MustRetry_ToSucceed(t *testing.T) {
	t.Parallel()

	var count uint

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	slowHandler := func(w http.ResponseWriter, r *http.Request) {
		if count == 2 {
			return
		}

		time.Sleep(1 * time.Second)
		count++
	}

	ts := httptest.NewServer(http.HandlerFunc(slowHandler))
	defer ts.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}

	if err := httpTest(ctx, client, ts.URL, 4); err != nil {
		t.Fatalf("should be a pasing request got err: %v", err)
	}
}

func Test_tcpTest(t *testing.T) {
	l, _ := net.Listen("tcp4", "127.0.0.1:0")
	defer l.Close()

	exit := make(chan struct{}, 1)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-exit:
					return
				default:
					log.Println("new connection")
					_, _ = conn.Write([]byte("LO" + "\n"))
					conn.Close()
				}
			}

		}
	}()

	localURL := fmt.Sprintf("tcp://%s", l.Addr().String())
	t.Log(localURL)

	gctx := context.Background()
	gctx, cancel := context.WithTimeout(gctx, 20*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		url     string
		retries int
		timeout time.Duration
		wantErr bool
	}{
		{"local working", localURL, 0, 500 * time.Millisecond, false},
		{"not routed network should die with global timeout", "tcp://203.0.113.0:80", 20, 10 * time.Second, true},
		{"not routed network should die with retries", "tcp://203.0.113.0:80", 0, 500 * time.Millisecond, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(gctx, 2*time.Second)
			tt := tt
			if err := tcpTest(ctx, tt.url, tt.timeout, tt.retries); (err != nil) != tt.wantErr {
				t.Errorf("tcpTest() error = %v, wantErr %v", err, tt.wantErr)
			}
			cancel()
		})
	}
	exit <- struct{}{}
}
