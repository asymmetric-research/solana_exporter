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
		server   *http.Server
		listener net.Listener
		// ResponseMap allows tests to set custom responses for specific methods
		ResponseMap map[string]any
		mu          sync.RWMutex
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
func NewMockServer() (*MockServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}

	ms := &MockServer{listener: listener, ResponseMap: make(map[string]any)}

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

// SetResponse sets a custom response for a specific method
func (ms *MockServer) SetResponse(method string, response any) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.ResponseMap[method] = response
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

	ms.mu.RLock()
	result, exists := ms.ResponseMap[req.Method]
	ms.mu.RUnlock()

	if !exists {
		// Fall back to default responses if no custom response is set
		result = getDefaultResponse(req.Method)
	}

	response := RPCResponse{JsonRPC: "2.0", Result: result, ID: req.Id}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Helper function to get default responses
func getDefaultResponse(method string) any {
	switch method {
	case "getBalance":
		return map[string]any{"context": map[string]any{"slot": 1234567}, "value": 100000000}
	case "getSlot":
		return 1234567
	// Add more default responses as needed
	default:
		return nil
	}
}

// TestHelper is a helper struct for tests
type TestHelper struct {
	Server *MockServer
	T      *testing.T
}

// NewTestHelper creates a new test helper with a running mock server
func NewTestHelper(t *testing.T) *TestHelper {
	server, err := NewMockServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Failed to close mock server: %v", err)
		}
	})

	return &TestHelper{Server: server, T: t}
}
