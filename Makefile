
.PHONY: test

test:
	sudo go test ./... -v -covermode=count
