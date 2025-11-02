proxs.darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o proxs.darwin-arm64 .


.PHONY: clean
clean:
	rm -f proxs.darwin-arm64
