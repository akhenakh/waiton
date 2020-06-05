package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/namsral/flag"
	"golang.org/x/sync/errgroup"
)

const (
	defaultGlobalTimeout = 1 * time.Minute
	defaultURLTimeout    = 10 * time.Second
)

func httpTest(ctx context.Context, client *http.Client, url string, maxRetries int) error {
	retries := 0

	for {
		if retries >= maxRetries {
			return errors.New("reached max number of retries")
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)

		if resp != nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}

			err = fmt.Errorf("request returned status %d", resp.StatusCode)
		}

		if err == nil {
			err = errors.New("no response")
		}

		retries++

		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("connection not finished in time error was: %w", err)
			}

			return errors.New("connection not finished in time")
		case <-time.After(1 * time.Second):
			continue
		}
	}
}

func tcpTest(ctx context.Context, url string, timeout time.Duration, maxRetries int) error {
	var d net.Dialer

	retries := 0

	for {
		if retries >= maxRetries {
			return errors.New("reached max number of retries")
		}

		lctx, cancel := context.WithTimeout(ctx, timeout)

		_, err := d.DialContext(lctx, "tcp", strings.TrimPrefix(url, "tcp://"))
		if err == nil {
			cancel()
			return nil
		}

		cancel()
		retries++

		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("connection not finished in time error was: %w", err)
			}

			return errors.New("connection not finished in time")
		case <-time.After(1 * time.Second):
			continue
		}
	}
}

func main() {
	var fs *flag.FlagSet

	prefix := os.Getenv("WAITON_PREFIX")
	if prefix != "" {
		fs = flag.NewFlagSetWithEnvPrefix(os.Args[0], prefix, 0)
	} else {
		fs = flag.NewFlagSet(os.Args[0], 0)
	}

	var (
		urlsString = fs.String(
			"urls",
			"",
			"comma separated urls to test, supported schemes are http:// https:// & tcp://",
		)

		globalTimeout = fs.Duration(
			"globalTimeout",
			defaultGlobalTimeout,
			"timeout to wait for all the hosts to be available before failure. (default 1mn)",
		)

		urlTimeout = fs.Duration("urlTimeout",
			defaultURLTimeout,
			"timeout to wait for one host to be available before retry. (default 10s)",
		)

		maxRetries = fs.Int("maxRetries",
			100,
			"max number of retries before giving up. (default 100)",
		)
	)

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal("can't parse args")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, *globalTimeout)

	defer cancel()

	var urls []string

	for _, u := range strings.Split(*urlsString, ",") {
		u = strings.TrimSpace(u)
		if len(u) > 0 {
			urls = append(urls, u)
		}
	}

	if len(urls) == 0 {
		log.Fatal("no url to test")
	}

	httpClient := &http.Client{Timeout: *urlTimeout}

	var g errgroup.Group

	for _, surl := range urls {
		su := surl

		u, err := url.Parse(su)
		if err != nil {
			log.Fatalf("can't parse url: %s error: %v", su, err)
		}

		switch u.Scheme {
		case "http", "https":
			g.Go(func() error {
				err := httpTest(ctx, httpClient, su, *maxRetries)
				if err != nil {
					log.Printf("%s error: %v", su, err)
				} else {
					log.Printf("%s completed\n", su)
				}
				return err
			})
		case "tcp":
			g.Go(func() error {
				err := tcpTest(ctx, su, *urlTimeout, *maxRetries)
				if err != nil {
					log.Printf("%s error: %v", su, err)
				} else {
					log.Printf("%s completed\n", su)
				}
				return err
			})
		default:
			log.Fatalf("unsupported scheme: %s", su)
		}
	}

	if err := g.Wait(); err == nil {
		log.Println("All tests completed")
	} else {
		os.Exit(1)
	}
}
