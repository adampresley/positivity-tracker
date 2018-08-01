.DEFAULT_GOAL := dev

prepare:
	go generate

build-windows: prepare
	GOOS=windows GOARCH=amd64 go build

build-mac: prepare
	GOOS=darwin GOARCH=amd64 go build

run:
	go run *.go

dev: prepare run
