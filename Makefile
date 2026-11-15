.PHONY: all build build-web build-go test test-race bench clean run demo

all: build

build-web:
	cd web && npm run build

build-go:
	go build -ldflags="-s -w" -o bin/claimguard.exe ./cmd/claimguard

build: build-web build-go

test:
	go test -v ./...

test-race:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem ./internal/...

run:
	./bin/claimguard.exe serve --config examples/claimguard.yaml

demo:
	./bin/claimguard.exe demo

clean:
	rm -rf bin/
	rm -rf web/dist/
