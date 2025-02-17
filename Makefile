.PHONY: build run test clean

build:
	go build -o bin/processor.exe cmd/processor/main.go

run: build
	bin\processor.exe

test:
	go test ./...

clean:
	if exist bin rmdir /s /q bin
	if exist tmp rmdir /s /q tmp

dev:
	air -c .air.toml