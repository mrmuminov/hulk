# HULK - HTTP Unbearable Load King

A DoS testing tool that generates massive HTTP load against a target server using raw TCP connections. Rewritten for maximum throughput — see benchmark below.

## Usage

```
./hulk -site http://target.com
```

Press `Ctrl+C` to stop.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-site` | `http://localhost` | Target URL |
| `-safe` | false | Stop when server returns 500 |
| `-data` | "" | POST body (switches to POST requests) |
| `-agents` | "" | File with custom User-Agent strings (one per line) |
| `-header` | | Custom header (can be used multiple times: `-header "X-Foo: bar"`) |
| `-version` | false | Print version and exit |

## Environment

- `HULKMAXPROCS` — max concurrent goroutines (default: 1023)

## Docker

```
docker build -t hulk -f docker/Dockerfile .
docker run -it hulk -site http://target.com
```

## Benchmark vs grafov/hulk

| Target | This version (raw TCP) | grafov/hulk (net/http) | Speedup |
|--------|----------------------|----------------------|---------|
| Localhost (HTTP) | ~189,000 req/s | ~13,100 req/s | **14x** |
| Localhost (slow server) | ~8,850 req/s | ~200 req/s | **44x** |
| Remote site (HTTP/HTTPS) | ~4,800-7,500 req/s | ~1,200 req/s | **4-6x** |
| Remote site (HTTPS only) | ~280-420 req/s | 0 (fails to connect) | — |

### Why is it faster?

The old version (`net/http`) for every request:
1. Creates a new `http.Client` (TLS handshake overhead)
2. Reads the response body then calls `.Body.Close()`
3. All goroutines report results through a single `chan` — **bottleneck**

This version (raw TCP):
1. Each goroutine opens **one TCP connection** and sends requests over keep-alive — TLS handshake only once
2. Does not read the response body — **zero-copy**
3. Pre-spawned goroutines, `sync/atomic` counters — **no channels, no bottleneck**
4. Each goroutine has its own PRNG (`rand.New(rand.NewSource(...))`) — **no mutex contention**
5. Direct `net.Dialer` + `net.TCPConn.Write()` — no pooling, no wrapping, no interface overhead

---

Based on [grafov/hulk](https://github.com/grafov/hulk) (Go port) and [Barry Shteiman's original HULK](http://www.sectorix.com/2012/05/17/hulk-web-server-dos-tool/) (Python).
