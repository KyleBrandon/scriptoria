build:
	@go build -o ./bin/scriptoria ./cmd/scriptoria

test:
	@go test ./...

run: build
	@./bin/scriptoria

rund: build
	@./bin/scriptoria -log_level=debug
