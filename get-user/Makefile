.PHONY: build clean test

build:
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
	zip function.zip bootstrap

clean:
	rm -f bootstrap function.zip

test:
	go test ./... 

copy:
	cp ./bootstrap $(ARTIFACTS_DIR)/.

build-GetUserFunction: clean build copy