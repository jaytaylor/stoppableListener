package stoppableListener

import (
	"net"
	"os/exec"
	"testing"
	"time"
)

func Test_StopSafely(t *testing.T) {
	var (
		acceptLoopDone = make(chan struct{})
		l, err         = net.Listen("tcp", "")
	)
	if err != nil {
		t.Fatal(err)
	}

	stoppable, err := New(l)
	if err != nil {
		t.Fatal(err)
	}
	stoppable.Verbose = true

	go func() {
		defer func() {
			acceptLoopDone <- struct{}{}
		}()
		for {
			// Listen for an incoming connection.
			conn, err := stoppable.Accept()
			if err != nil {
				if err == StoppedError {
					t.Log("detected listener socket stop, accept loop exiting")
					return
				}
				t.Logf("error accepting connection: %s", err)
				continue
			}
			// Handle connections in a new goroutine.
			go conn.Close()
		}
	}()

	addr := stoppable.TCPListener.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Error splitting host:port from address %q: %s", addr, err)
	}

	output, err := exec.Command("nc", "-w", "1", host, port).CombinedOutput()
	if err != nil {
		t.Fatalf("netcat port connect test failed: %s (output=%v)", err, string(output))
	}

	if err := stoppable.StopSafely(); err != nil {
		t.Errorf("StopSafely error: %s", err)
	}

	timeout := 2 * time.Second

	select {
	case <-acceptLoopDone:
	case <-time.After(timeout):
		t.Errorf("Timed out after %s waiting for accept loop exit signal: accept loop didn't exit after the listener was stopped", timeout)
	}
}
