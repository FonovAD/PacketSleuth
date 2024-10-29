.PHUNY: build
build: 
	go build -v ./cmd/main.go

.DEFAULT_GOAL := build