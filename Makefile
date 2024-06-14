.PHONY: test test-rpc

test:
	@go test ./cmd/solana_exporter -v -short

test-rpc:
	@go test ./pkg/rpc -v
