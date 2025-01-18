build:
	@go build -o ./bin/scriptoria ./cmd/scriptoria

test:
	@go test ./...

run: build
	@./bin/scriptoria


