viamrosmodule: cmd/module/cmd.go
	go build -o rosmodule cmd/module/cmd.go

viamrosmoduleaarch64: cmd/module/cmd.go
	env GOARCH=arm64 GOOS=linux go build -o rosmodule cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .
