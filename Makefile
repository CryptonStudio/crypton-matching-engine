test:
	go test -v -count=1 ./...

test100:
	go test -v -count=100 ./...

race:
	go test -v -race -count=1 ./...

gen: 
	go generate ./...

.PHONY: cover
cover:
	go test -short -count=1 -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	rm coverage.out

fuzz:
	cd matching/tests && go clean -fuzzcache && go test -fuzz FuzzAllOrders

fuzz-chain:
	cd matching/tests && go clean -fuzzcache && go test -fuzz FuzzChainOrders

test-mem:
	cd matching/tests && go test -run TestAllocatorCollisions