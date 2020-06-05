package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/context"
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
