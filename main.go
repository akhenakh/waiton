package main

import (
	"context"
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

const appName = "waiton"

var (
	version = "no version from LDFLAGS"
)

func httpTest(ctx context.Context, client *http.Client, url string, maxRetries int) error {
	retries := 0

	for {
		if retries >= maxRetries {
			return fmt.Errorf("reached max number of retries")
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err == nil {
			return nil
		}

		if resp != nil {
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			err = fmt.Errorf("request returned status %d", resp.StatusCode)
		}

		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("connection not finished in time error was: %s", err)
			}
			return fmt.Errorf("connection not finished in time")
		case <-time.After(1 * time.Second):
			continue
		}
		retries++
	}

	return nil
}

func tcpTest(ctx context.Context, url string, timeout time.Duration, maxRetries int) error {
	var d net.Dialer

	retries := 0

	for {
		if retries >= maxRetries {
			return fmt.Errorf("reached max number of retries")
		}
		lctx, cancel := context.WithTimeout(ctx, timeout)
		_, err := d.DialContext(lctx, "tcp", strings.TrimPrefix(url, "tcp://"))
		if err == nil {
			cancel()
			return nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("connection not finished in time error was: %s", err)
			}
			return fmt.Errorf("connection not finished in time")
		case <-time.After(1 * time.Second):
			continue
		}
		cancel()
		retries++
	}

	return nil
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
		urlsString    = fs.String("urls", "", "comma separated urls to test, supported schemes are http:// & tcp://")
		globalTimeout = fs.Duration("globalTimeout", time.Duration(1*time.Minute), "timeout to wait for all the hosts to be available before failure. (default 1mn)")
		urlTimeout    = fs.Duration("urlTimeout", time.Duration(10*time.Second), "timeout to wait for one host to be available before retry. (default 10s)")
		maxRetries    = fs.Int("maxRetries", 100, "max number of retries before giving up. (default 100)")
	)

	fs.Parse(os.Args[1:])

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, *globalTimeout)
	defer cancel()

	var urls []string
	for _, u := range strings.Split(*urlsString, ",") {
		urls = append(urls, strings.TrimSpace(u))
	}

	httpClient := http.DefaultClient
	httpClient.Timeout = *urlTimeout

	var g errgroup.Group

	for _, surl := range urls {
		su := surl
		u, err := url.Parse(su)
		if err != nil {
			log.Fatalf("can't parse url: %s error: %v", su, err)
		}

		switch u.Scheme {
		case "http":
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
		os.Exit(2)
	}
}
