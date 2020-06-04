# waiton

A very simple CLI tool to wait for TCP or HTTP server to respond.

You can use environment variable or CLI flags as follow:

- `GLOBALTIMEOUT` `-globalTimeout` 
  Timeout to wait for all the hosts to be available before failure, default `1m`.
 
- `URLTIMEOUT` `-urlTimeout`
  Timeout to wait for one host to be available before retry, default `10s`.

- `URLS` `-urls`
  comma separated list of urls to test, supported schemes are `http://` & `tcp://`


## Example

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

For HTTP & TCP, waiton wll retry every 1s as a backoff strategy.

