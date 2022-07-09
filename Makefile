.PHONY: proto
proto:
	@protoc -I ./third_party -I ./proto --go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		--validate_out=proto --validate_opt=paths=source_relative,lang=go \
		proto/*.proto

.PHONY: proto
inject:
	@protoc-go-inject-tag -input=proto/*.pb.go

#generate dependency using wire
.PHONY: wire
wire:
	cd ezjob && wire
