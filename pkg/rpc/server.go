package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

type (
	// MockServer represents a mock Solana RPC server for testing
	MockServer struct {
		// Responses allows tests to set custom responses for specific methods
		Responses map[string]any
		server    *http.Server
		listener  net.Listener
		mu        sync.RWMutex
	}

	// RPCResponse represents a JSON-RPC response
	RPCResponse struct {
		JsonRPC string       `json:"jsonrpc"`
		Result  any          `json:"result"`
		Error   rpc.RPCError `json:"error,omitempty"`
		ID      any          `json:"id"`
	}
)

// NewMockServer creates a new mock server instance
func NewMockServer(responses map[string]any) (*MockServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}

	ms := &MockServer{listener: listener, Responses: responses}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handleRPCRequest)

	ms.server = &http.Server{Handler: mux}

	go func() {
		_ = ms.server.Serve(listener)
	}()

	return ms, nil
}

// URL returns the URL of the mock server
func (ms *MockServer) URL() string {
	return fmt.Sprintf("http://%s", ms.listener.Addr().String())
}

// Close shuts down the mock server
func (ms *MockServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ms.server.Shutdown(ctx)
}

func (ms *MockServer) MustClose() {
	if err := ms.Close(); err != nil {
		panic(err)
	}
}

// SetResponse sets a custom response for a specific method
func (ms *MockServer) SetResponse(method string, response any) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Responses[method] = response
}

func (ms *MockServer) handleRPCRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req rpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := RPCResponse{JsonRPC: "2.0", ID: req.Id}
	ms.mu.RLock()
	result, exists := ms.Responses[req.Method]
	ms.mu.RUnlock()

	if !exists {
		// Fall back to default responses if no custom response is set
		response.Error = rpc.RPCError{
			Code:    -32601,
			Message: "Method not found",
		}
	} else {
		response.Result = result
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TestHelper is a helper struct for tests
type TestHelper struct {
	Server *MockServer
	Client *rpc.Client
	T      *testing.T
}

// NewTestHelper creates a new test helper with a running mock server
func NewTestHelper(t *testing.T, responses map[string]any) *TestHelper {
	server, err := NewMockServer(responses)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Failed to close mock server: %v", err)
		}
	})

	return &TestHelper{Server: server, Client: rpc.NewRPCClient(server.URL(), time.Second), T: t}
}
