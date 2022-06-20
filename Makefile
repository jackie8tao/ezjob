.PHONY: proto
proto:
	buf generate

.PHONY: wire
wire:
	cd utils/wireutil && wire
