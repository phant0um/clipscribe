.PHONY: test vet lint build
test:
	go test ./...
vet:
	go vet ./...
lint:
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
build:
	go build -o clipscribe ./cmd/clipscribe

fuzz:
	go test -run ^$$ -fuzz FuzzParseURL -fuzztime 30s ./internal/media
	go test -run ^$$ -fuzz FuzzFileName -fuzztime 30s ./internal/vault
	go test -run ^$$ -fuzz FuzzRenderMarkdown -fuzztime 30s ./internal/transcript
