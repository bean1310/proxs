proxs.darwin-arm64: *.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o proxs.darwin-arm64 .

.PHONY: test test-verbose test-cover test-bench clean
test:
	go test ./...

test-verbose:
	go test -v ./...

test-cover:
	go test -cover ./...

test-bench:
	go test -bench=. ./...

.PHONY: clean
clean:
	rm -f proxs.darwin-arm64
