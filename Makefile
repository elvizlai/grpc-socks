idl:
	rm -rf pb/*.pb.go
	protoc -I=. pb/*.proto --go_out=plugins=grpc:.

bin:
	rm -rf exec_bin
	mkdir exec_bin
	go build -ldflags "-s -w" -o ./exec_bin/client ./client/*.go
	go build -ldflags "-s -w" -o ./exec_bin/server ./server/*.go
