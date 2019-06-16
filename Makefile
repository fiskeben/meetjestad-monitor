PHONY: clean

meetjebatterij:$(shell find . -name "*.go")
	go mod download
	go build .

dist:
	go mod download
	GOOS=linux go build -tags netgo -ldflags '-w -s' -o meetjebatterij-linux .

clean:meetjebatterij
	rm meetjebatterij
