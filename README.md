# HULK - HTTP Unbearable Load King

DoS testing tool. Sends massive HTTP load to a target server using raw TCP connections.

## Usage

```
./hulk -site http://target.com
```

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
