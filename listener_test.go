package stoppableListener

import (
	"net"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestStop(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	stoppable, err := New(l)
	if err != nil {
		t.Fatal(err)
	}
	stoppable.Verbose = true

	runScenario(t, stoppable, stoppable.Stop, false)
}

func TestStopSafely(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	stoppable, err := New(l)
	if err != nil {
		t.Fatal(err)
	}
	stoppable.Verbose = true

	runScenario(t, stoppable, stoppable.StopSafely, true)
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func runScenario(t *testing.T, stoppable *StoppableListener, stopperFunc func() error, stopperBlocksUntilDone bool) {
	acceptLoopDone := make(chan struct{})

	go func() {
		defer func() {
			acceptLoopDone <- struct{}{}
		}()
		for {
			// Listen for an incoming connection.
			conn, err := stoppable.Accept()
			if err != nil {
				if err == StoppedError {
					t.Log("Detected listener socket stop, accept loop exiting")
					return
				}
				t.Logf("Error accepting connection: %s", err)
				continue
			}
			// Handle connections in a new goroutine.
			go conn.Close()
		}
	}()

	addr := stoppable.TCPListener.Addr().String()

	if _, _, err := net.SplitHostPort(addr); err != nil {
		t.Fatalf("Error splitting host:port from address %q: %s", addr, err)
	}

	if conn, err := net.DialTimeout("tcp", addr, stoppable.StopCheckTimeout); err != nil {
		t.Errorf("Unexpected connection failure to TCP listener at address=%s: %s", addr, err)
	} else {
		if err = conn.Close(); err != nil {
			t.Error(err)
		}
	}

	if err := stopperFunc(); err != nil {
		t.Errorf("Error: stopperFunc()=%s stopperBlocksUntilDone=%s error=%s", getFunctionName(stopperFunc), stopperBlocksUntilDone, err)
	}

	if stopperBlocksUntilDone {
		if conn, err := net.DialTimeout("tcp", addr, stoppable.StopCheckTimeout); err != nil {
			t.Logf("Received expected connection rejection after %s() to TCP listener at address=%s: %s", getFunctionName(stopperFunc), addr, err)
		} else {
			if err = conn.Close(); err != nil {
				t.Error(err)
			}
		}
	} else {
		if err := stoppable.waitUntilStopped(); err != nil {
			t.Error(err)
		}
	}

	if conn, err := net.DialTimeout("tcp", addr, stoppable.StopCheckTimeout); err != nil {
		t.Logf("Received expected connection rejection after %s() to TCP listener at address=%s: %s", getFunctionName(stopperFunc), addr, err)
	} else {
		if err = conn.Close(); err != nil {
			t.Error(err)
		}
	}

	timeout := 2 * time.Second

	select {
	case <-acceptLoopDone:
	case <-time.After(timeout):
		t.Errorf("Timed out after %s waiting for accept loop exit signal: accept loop didn't exit after the listener was stopped", timeout)
	}
}
