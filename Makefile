.PHONY: dev build clean

all: dev

dev: build
	./todo

build: clean
	go get ./...
	go build .

test:
	go test ./...

clean:
	rm -rf todo
