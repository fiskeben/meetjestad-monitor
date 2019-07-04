PHONY: clean dist test

meetjestad-monitor:$(shell find . -name "*.go")
	go mod download
	go build -mod vendor .

test:
	go test ./...

dist:
	go mod download
	GOOS=linux go build -tags netgo -ldflags '-w -s' -o meetjestad-monitor-linux .

clean:meetjestad-monitor
	rm meetjestad-monitor*
