PROTO_SRC=protocol/proto
PROTO_DEST=protocol/pb
PWD = $(shell pwd)
export GOBIN=$(PWD)/../bin

all:proto gen install

install:
	go install ./gamesvr ./router ./gamegate

proto:
	@echo "gen proto ..."
	@test -d $(PROTO_DEST) || mkdir -p $(PROTO_DEST)
	protoc -I=$(PROTO_SRC) --go_out=$(PROTO_DEST) $(PROTO_SRC)/*.proto
	@echo "gen proto ok"

gen:
	go generate ./errcode ./protocol

clean:
	#go clean
	rm protocol/pb/*.pb.go
	echo clean pb.go ok