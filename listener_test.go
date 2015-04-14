package stoppableListener

import (
	"fmt"
	"net"
	"testing"
)

const port = 31337

func Test_StopSafely(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:"+fmt.Sprint(port))
	if err != nil {
		t.Fatal(err)
	}

	stoppable, err := New(l)
	stoppable.Verbose = true
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			// Listen for an incoming connection.
			conn, err := stoppable.Accept()
			if err != nil {
				if err == StoppedError {
					t.Log("socket stopped, listener loop exiting")
					return
				}
				t.Logf("error accepting connection: %s", err)
				continue
			}
			// Handle connections in a new goroutine.
			go conn.Close()
		}
	}()

	if err := stoppable.StopSafely(); err != nil {
		t.Fatal(err)
	}
}
