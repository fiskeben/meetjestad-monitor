PHONY: clean

meetjebatterij:$(shell find . -name "*.go")
	go mod download
	go build .

dist:
	GOOS=linux go build -o meetjebatterij-linux .

clean:meetjebatterij
	rm meetjebatterij