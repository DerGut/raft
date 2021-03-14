build:
	go build ./cmd/raft/main.go
	
build-race:
	go build --race ./cmd/raft/main.go

build-client:
	go build -o client ./cmd/client/main.go

run1:
	./main --address 127.0.0.1:3000 --members 127.0.0.1:3001,127.0.0.1:3002 --state "tmp/s1"

run2:
	./main --address 127.0.0.1:3001 --members 127.0.0.1:3000,127.0.0.1:3002 --state "tmp/s2"

run3:
	./main --address 127.0.0.1:3002 --members 127.0.0.1:3000,127.0.0.1:3001 --state "tmp/s3"

