# stoppableListener

Go package for graceful shutdown of TCP socket listeners.

Works with go 1.4+.

[![Documentation](https://godoc.org/github.com/jaytaylor/stoppableListener?status.svg)](https://godoc.org/github.com/jaytaylor/stoppableListener)
[![Build Status](https://travis-ci.org/jaytaylor/stoppableListener.svg?branch=master)](https://travis-ci.org/jaytaylor/stoppableListener)
[![Report Card](https://goreportcard.com/badge/github.com/jaytaylor/stoppableListener)](https://goreportcard.com/report/github.com/jaytaylor/stoppableListener)

## About

A cleanly stoppable TCP listener in Go. This library wraps an existing TCP connection object.  A goroutine calling `Accept()` is interrupted with `StoppedError` whenever the listener is stopped by a call to `Stop()`.

For a fully functional example see [example/example.go](example/example.go).

Quick usage overview:

```go
	originalListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	sl, err := stoppableListener.New(originalListener)
	if err != nil {
		panic(err)
	}
```

## Requirements

* Go version 1.4 or newer
* Linux, Mac OS X, or Windows

## Running the tests

    go test -v

## Notes

### Graceful shutdown in Go 1.8+

Go 1.8 added built-in support for graceful shutdown of HTTP servers.

See also: [Runnable example](https://gist.github.com/peterhellberg/38117e546c217960747aacf689af3dc2#gistcomment-1982608)

