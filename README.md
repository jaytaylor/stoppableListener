# stoppableListener

[![Documentation](https://godoc.org/github.com/jaytaylor/stoppableListener?status.svg)](https://godoc.org/github.com/jaytaylor/stoppableListener)
[![Build Status](https://travis-ci.org/jaytaylor/stoppableListener.svg?branch=master)](https://travis-ci.org/jaytaylor/stoppableListener)
[![Report Card](https://goreportcard.com/badge/github.com/jaytaylor/stoppableListener)](https://goreportcard.com/report/github.com/jaytaylor/stoppableListener)

## About

An example of a stoppable TCP listener in Go. This library wraps an existing TCP connection object. A goroutine calling `Accept()`
is interrupted with `StoppedError` whenever the listener is stopped by a call to `Stop()`. Usage is demonstrated below, and in `example/example.go`.

```
	originalListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	sl, err := stoppableListener.New(originalListener)
	if err != nil {
		panic(err)
	}
```

## Running the tests

    go test .

