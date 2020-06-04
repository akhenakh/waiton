# waiton

A very simple CLI tool to wait for TCP or HTTP server to respond.

You can use environment variable or CLI flags as follow:

- `GLOBALTIMEOUT` `-globalTimeout` 
  Timeout to wait for all the hosts to be available before failure, default `30s`.
 
- `URLTIMEOUT` `-urlTimeout`
  Timeout to wait for one host to be available before retry, default `5s`.

- `URLS` `-urls`
  comma separated list of urls to test, supported schemes are `http://` & `tcp://`


## Exanple

```
GLOBALTIMEOUT=10s URLS=""http://www.google.com,tcp://localhost:22" ./waiton
```

```
./waiton -globalTimeout=10s -urls="URLS=http://www.google.com,tcp://localhost:22"
```

```
docker run  --rm -it -e URLS="http://www.goole.com" akhenak/waiton:latest
```

## Details

For HTTP, waiton is using [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) as an HTTP client, using the retries & backoff strategies.

For TCP, waiton is using a simple 1s sleep between retries.
