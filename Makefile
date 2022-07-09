.PHONY: proto
proto:
	buf generate

#generate dependency using wire
.PHONY: wire
wire:
	cd ezjob && wire
