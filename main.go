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

	"github.com/hashicorp/go-retryablehttp"
	"github.com/namsral/flag"
	"golang.org/x/sync/errgroup"
)

const appName = "waiton"

var (
	version = "no version from LDFLAGS"
)

func httpTest(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request returned status %d", resp.StatusCode)
	}

	return nil
}

func tcpTest(ctx context.Context, url string, timeout time.Duration) error {
	var d net.Dialer

	for {
		lctx, cancel := context.WithTimeout(ctx, timeout)
		_, err := d.DialContext(lctx, "tcp", strings.TrimPrefix(url, "tcp://"))
		if err == nil {
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
	}

	return nil
}

func main() {
	var fs *flag.FlagSet

	prefix := os.Getenv("PREFIX")
	if prefix != "" {
		fs = flag.NewFlagSetWithEnvPrefix(os.Args[0], prefix, 0)
	} else {
		fs = flag.NewFlagSet(os.Args[0], 0)
	}

	var (
		urlsString    = fs.String("urls", "", "comma separated urls to test, supported schemes are http:// & tcp://")
		globalTimeout = fs.Duration("globalTimeout", time.Duration(30*time.Second), "timeout to wait for all the hosts to be available before failure. (default 30s)")
		urlTimeout    = fs.Duration("urlTimeout", time.Duration(5*time.Second), "timeout to wait for one host to be available before retry. (default 5s)")
	)

	fs.Parse(os.Args[1:])

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, *globalTimeout)
	defer cancel()

	var urls []string
	for _, u := range strings.Split(*urlsString, ",") {
		urls = append(urls, strings.TrimSpace(u))
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil
	retryClient.RetryMax = 20
	httpClient := retryClient.StandardClient()
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
				err := httpTest(ctx, httpClient, su)
				if err != nil {
					log.Printf("%s error: %v", su, err)
				} else {
					log.Printf("%s completed\n", su)
				}
				return err
			})
		case "tcp":
			g.Go(func() error {
				err := tcpTest(ctx, su, *urlTimeout)
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
