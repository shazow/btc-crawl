all: btc-crawl

**/*.go:
	go build ./...

btc-crawl: **/*.go *.go
	go build .

build: btc-crawl

clean:
	rm btc-crawl

run: btc-crawl
	./btc-crawl -v
