package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/slog"
	"go.uber.org/zap"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

type (
	// MockServer represents a mock Solana RPC server for testing
	MockServer struct {
		easyResults map[string]any
		server      *http.Server
		listener    net.Listener
		mu          sync.RWMutex
		logger      *zap.SugaredLogger
	}
)

// NewMockServer creates a new mock server instance
func NewMockServer(easyResults map[string]any) (*MockServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}

	ms := &MockServer{listener: listener, easyResults: easyResults, logger: slog.Get()}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handleRPCRequest)

	ms.server = &http.Server{Handler: mux}

	go func() {
		_ = ms.server.Serve(listener)
	}()

	return ms, nil
}

// URL returns the URL of the mock server
func (s *MockServer) URL() string {
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}

// Close shuts down the mock server
func (s *MockServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *MockServer) MustClose() {
	if err := s.Close(); err != nil {
		panic(err)
	}
}

// SetEasyResult sets a custom response for a specific method
func (s *MockServer) SetEasyResult(method string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.easyResults[method] = result
}

func (s *MockServer) GetResult(method string) (any, *RPCError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.easyResults[method]
	if !ok {
		return nil, &RPCError{Code: -32601, Message: "Method not found"}
	}
	return result, nil
}

func (s *MockServer) handleRPCRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := Response[any]{Jsonrpc: "2.0", Id: request.Id}
	result, rpcErr := s.GetResult(request.Method)
	if rpcErr != nil {
		response.Error = *rpcErr
	} else {
		response.Result = result
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// NewTestClient creates a new test helper with a running mock server
func NewTestClient(t *testing.T, easyResponses map[string]any) (*MockServer, *Client) {
	server, err := NewMockServer(easyResponses)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Failed to close mock server: %v", err)
		}
	})

	client := NewRPCClient(server.URL(), time.Second)
	return server, client
}
