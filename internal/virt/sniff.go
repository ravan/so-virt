package virt

import (
	"bytes"
	"fmt"
	"io"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	DumpHttpRequest bool
)

type loggingRoundTripper struct {
	transport http.RoundTripper
	dumpDir   string
}

// ensureDumpDir ensures the dump directory exists.
func mustEnsureDumpDir(dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
}

// writeLogToFile writes the given content to a uniquely named file.
func writeLogToFile(dumpDir, prefix, content string) error {
	timestamp := time.Now().Format("20060102_150405.000000") // Format: YYYYMMDD_HHMMSS.mmmmmm
	filename := fmt.Sprintf("%s_%s.log", prefix, timestamp)
	filePath := filepath.Join(dumpDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func (lrt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	var reqBody []byte
	var err error
	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Restore body after reading
	}

	reqLog := fmt.Sprintf("Request:\n%s %s\nHeaders: %v\nBody: %s\n\n", req.Method, req.URL, req.Header, string(reqBody))

	if err := writeLogToFile(lrt.dumpDir, "request", reqLog); err != nil {
		return nil, err
	}

	// Perform request
	resp, err := lrt.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Log response
	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // Restore body after reading
	}
	respLog := fmt.Sprintf("Response:\nStatus: %s\nHeaders: %v\nBody: %s\n\n",
		resp.Status, resp.Header, string(respBody))

	if err := writeLogToFile(lrt.dumpDir, "response", respLog); err != nil {
		return nil, err
	}

	return resp, nil
}

func intercept(config *rest.Config) {
	if !DumpHttpRequest {
		return
	}
	mustEnsureDumpDir("http_requests")
	// Wrap the transport with our loggingRoundTripper
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &loggingRoundTripper{transport: rt, dumpDir: "http_requests"}
	}
}
