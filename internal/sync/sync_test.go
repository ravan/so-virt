package sync

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/ravan/so-virt/internal/config"
	"github.com/ravan/so-virt/internal/virt"
	"github.com/ravan/stackstate-client/stackstate/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSync(t *testing.T) {
	//Dump the http requests to server if you need to debug.
	api.DumpHttpRequest = false
	virt.DumpHttpRequest = false

	// Use the following config for connecting to live server
	//conf := getConfig(t)

	//Mock Server
	conf, server := getMockConf(t)
	defer server.Close()

	factory, err := Sync(conf)
	require.NoError(t, err)
	assert.Equal(t, 24, factory.GetComponentCount())
	assert.Equal(t, 22, factory.GetRelationCount())
}

// -- Testing Infrastructure --

var opts = &slog.HandlerOptions{Level: slog.LevelDebug}
var handler = slog.NewJSONHandler(os.Stdout, opts)
var logger = slog.New(handler)

func init() { slog.SetDefault(logger) }

func getMockConf(t *testing.T) (*config.Configuration, *httptest.Server) {
	conf := getConfig(t)
	conf.Kubernetes.KubeConfig = "../../testdata/mock-kubeconfig.yaml"
	server := getMockServer(t, getMockHandler(t))
	return conf, server
}

func getMockServer(t *testing.T, hf http.HandlerFunc) *httptest.Server {
	// create a listener with the desired port.
	l, err := net.Listen("tcp", "127.0.0.1:8182")
	require.NoError(t, err)
	ts := httptest.NewUnstartedServer(hf)
	require.NoError(t, err)

	// NewUnstartedServer creates a listener. Close that listener and replace
	// with the one we created.
	err = ts.Listener.Close()
	require.NoError(t, err)
	ts.Listener = l

	// Start the server.
	ts.Start()
	return ts
}

func getMockHandler(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "apis/kubevirt.io/v1/virtualmachineinstances") {
			loadRespFile(w, "apis/kubevirt.io/v1/virtualmachineinstances/response.json")
		} else {
			assert.Fail(t, fmt.Sprintf("unexpected request: %s", r.URL.Path))
		}
	}
}

func loadRespFile(w http.ResponseWriter, path string) {
	path = fmt.Sprintf("../../testdata/%s", path)
	_, err := os.Stat(path)
	if err == nil {
		if strings.HasSuffix(path, "json") {
			w.Header().Set("Content-Type", "application/json")
		}
		file, err := os.ReadFile(path)
		if err == nil {
			_, err := w.Write(file)
			if err == nil {
				return
			}
		}
	}
	slog.Error("file not found", "path", path)
	w.WriteHeader(http.StatusNotFound)
}

func getConfig(t *testing.T) *config.Configuration {
	require.NoError(t, os.Setenv("CONFIG_FILE", "../../conf.yaml"))
	require.NoError(t, godotenv.Load("../../.env"))
	require.NoError(t, os.Setenv("suseobservability.api_url", os.Getenv("SO_URL")))
	require.NoError(t, os.Setenv("suseobservability.api_key", os.Getenv("SO_API_KEY")))
	require.NoError(t, os.Setenv("suseobservability.api_token", os.Getenv("SO_TOKEN")))

	c, err := config.GetConfig()
	require.NoError(t, err)
	return c
}
