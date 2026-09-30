BIN := bin/strava_blamer

.PHONY: build test

build:
	go build -o $(BIN) ./cmd/strava_blamer

test:
	go test ./...
