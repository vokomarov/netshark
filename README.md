# netshark
[![Build Status](https://travis-ci.org/vokomarov/netshark.svg?branch=master)](https://travis-ci.org/vokomarov/netshark)
[![Coverage Status](https://coveralls.io/repos/github/vokomarov/netshark/badge.svg?branch=master)](https://coveralls.io/github/vokomarov/netshark?branch=master)
[![GoDoc](https://godoc.org/github.com/DimitarPetrov/stegify?status.svg)](https://godoc.org/github.com/DimitarPetrov/stegify)
[![Go Report Card](https://goreportcard.com/badge/github.com/vokomarov/netshark)](https://goreportcard.com/report/github.com/vokomarov/netshark)

## Overview

`netshark` is a simple command line tool to scan your local network to available hosts and his opened ports.

Features:
- Scan local network neighbor hosts
- Display client MAC address

Coming soon:
- Scan opened TCP ports
- Specify scanning port ranges
- Use predefined port ranges to scan
- Multithreaded scan
- Shadow scan
- Use STDIN as source of hosts to support Unix pipes

## Installation

#### Installing from Source

```
go get -u github.com/vokomarov/netshark
```

#### Download

Download binary for your system [here](https://github.com/vokomarov/netshark/releases).

## Usage

### As a command line tool

```shell script
# Scan local network for neighbor hosts
$ netshark scan hosts [--timeout 15]

# Scan given host for opened ports of all port range
$ netshark scan ports --host 192.168.1.1 # scan specific

# Specify TCP port range from lower to higher number of ports
$ netshark scan ports --range 1,65535
$ netshark scan ports --range-min 1024 # check registered ports and private ports range
$ netshark scan ports --range-max 1023 # check only well-known port range

# Or use predefined port ranges
$ netshark scan ports --range known   # scan ports from 0 to 1023
$ netshark scan ports --range reg     # scan ports from 1024 to 49151
$ netshark scan ports --range private # scan ports from 49152 to 65535
```

### Programmatically in your code

`netshark` can be used programmatically. You can visit [godoc](https://godoc.org/github.com/vokomarov/netshark) to check API documentation.

## License

`netshark` is open-sourced software licensed under the [MIT license](http://opensource.org/licenses/MIT).
