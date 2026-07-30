.PHONY: build run

build:
	go build -o transcriber .

run:
	go run . $(ARGS)
