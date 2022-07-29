BINARY_NAME=figgen

build:
	GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME} main.go

clean:
	go clean
	rm ${BINARY_NAME}