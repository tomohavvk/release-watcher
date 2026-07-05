.PHONY: build build-go frontend run dev clean test vet tidy

BINARY=./tmp/release-watcher

build: frontend build-go

build-go:
	go build -o $(BINARY) ./cmd/server/

frontend:
	cd frontend && npm install && npm run build

build:
	$(BINARY)

run:
	air

clean:
	rm -rf tmp/ webapp/dist/ frontend/node_modules/

test:
	go test -v -count=1 ./...

vet:
	go vet ./...

tidy:
	go mod tidy
