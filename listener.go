package stoppableListener

import (
	"errors"
	"fmt"
	// "log"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type StoppableListener struct {
	*net.TCPListener              // Wrapped listener.
	stop                 chan int // Channel used only to indicate listener should shutdown.
	MaxStopChecks        int      // Maximum number of stop checks before StopSafely() gives up and returns an error.
	StopCheckWaitSeconds int      // Number of seconds to wait for during each stop check.  Must be an integer gte 1, otherwise the resulting behavior is undefined.
}

var (
	DefaultMaxStopChecks        = 10 // Default stop check limit before error.
	DefaultStopCheckWaitSeconds = 1  // Default number of seconds to wait for during each check.

	StoppedError              = errors.New("listener stopped")
	ListenerWrapError         = errors.New("cannot wrap listener")
	NotStoppedError           = errors.New("listener failed to stop, port is still open after MaxStopChecks exceeded")
	PlatformNotSupportedError = errors.New("platform not supported")
)

// New creates a new stoppable TCP listener.
func New(l net.Listener) (*StoppableListener, error) {
	tcpL, ok := l.(*net.TCPListener)

	if !ok {
		return nil, ListenerWrapError
	}

	retval := &StoppableListener{
		tcpL,
		make(chan int),
		DefaultMaxStopChecks,
		DefaultStopCheckWaitSeconds,
	}

	return retval, nil
}

func (sl *StoppableListener) Accept() (net.Conn, error) {
	for {
		// Wait up to one second for a new connection.
		sl.SetDeadline(time.Now().Add(time.Second))

		newConn, err := sl.TCPListener.Accept()

		// Check for the channel being closed.
		select {
		case <-sl.stop:
			// log.Println("StoppableListener stop channel is closed")
			return nil, StoppedError

		default:
			// If the channel is still open, continue as normal.
			// log.Println("StoppableListener stop channel is open")
		}

		if err != nil {
			netErr, ok := err.(net.Error)

			// If this is a timeout, then continue to wait for
			// new connections.
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			}
		}

		return newConn, err
	}
}

func (sl *StoppableListener) Stop() {
	close(sl.stop)
}

// StopSafely waits until the socket is longer reachable, or returns an error
// if the check times out.
func (sl *StoppableListener) StopSafely() (err error) {
	sl.Stop()
	if err = sl.waitUntilStopped(); err != nil {
		return
	}
	return
}

// waitUntilStopped uses netcat (nc) to determine if the listening port is
// still accepting connections.  Returns nil when connections are no longer
// being accepted, or returns NotStoppedError if MaxStopChecks are exceeded.
//
// NB: This probably only works on *nix (i.e. NOT Windows).
func (sl *StoppableListener) waitUntilStopped() error {
	if runtime.GOOS == "windows" {
		return PlatformNotSupportedError
	}
	args := append([]string{"-v", "-w", fmt.Sprint(sl.StopCheckWaitSeconds)}, strings.Split(sl.TCPListener.Addr().String(), ":")...)
	for i := 0; i < sl.MaxStopChecks; i++ {
		/*out*/ _, err := exec.Command("nc", args...).CombinedOutput()
		if err != nil { // If `nc` exits with non-zero status code then that means the port is closed.
			return nil
		}
		/*log.Printf("waitUntilStopped nc output=%s\n", string(out))*/
	}
	// log.Println("waitUntilStopped max checks exceeded, stop failed")
	return NotStoppedError
}
