
.PHONY: test

test:
	sudo go test ./scanner/... -v -covermode=count
