BINARY_NAME=figgen

UNAME := $(shell uname)
ifeq ($(UNAME), Darwin)
TARGET = darwin
else
TARGET = linux
endif

build:
	GOARCH=amd64 GOOS=${TARGET} go build -o ${BINARY_NAME} cmd/figgen/main.go

clean:
	go clean
	rm ${BINARY_NAME}