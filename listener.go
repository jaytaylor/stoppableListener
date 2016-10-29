package stoppableListener

// Many thanks to Richard Crowley for writing http://rcrowley.org/articles/golang-graceful-stop.html.

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"time"
)

type StoppableListener struct {
	*net.TCPListener                   // Wrapped listener.
	stopCh               chan struct{} // Channel used only to indicate listener should shutdown.
	MaxStopChecks        int           // Maximum number of stop checks before StopSafely() gives up and returns an error.
	StopCheckWaitSeconds int           // Number of seconds to wait for during each stop check.  Must be an integer gte 1, otherwise the resulting behavior is undefined.
	Verbose              bool          // Activates verbose logging.
}

var (
	DefaultMaxStopChecks        = 3     // Default stop check limit before error.
	DefaultStopCheckWaitSeconds = 1     // Default number of seconds to wait for during each check.
	DefaultVerbose              = false // Default value for Verbose field of new StoppableListeners.

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

	sl := &StoppableListener{
		TCPListener:          tcpL,
		stopCh:               make(chan struct{}),
		MaxStopChecks:        DefaultMaxStopChecks,
		StopCheckWaitSeconds: DefaultStopCheckWaitSeconds,
		Verbose:              DefaultVerbose,
	}

	return sl, nil
}

func (sl *StoppableListener) Accept() (net.Conn, error) {
	for {
		// Wait up to one second for a new connection.
		sl.SetDeadline(time.Now().Add(time.Second))

		newConn, err := sl.TCPListener.Accept()

		if err != nil {
			// Check for stop request.
			select {
			case <-sl.stopCh:
				close(sl.stopCh)
				sl.stopCh = nil
				return nil, StoppedError
			default:
				// If no stop has been requested proceed with normal operation.
			}

			// If this is a timeout, then continue to wait for
			// new connections.
			if netErr, ok := err.(net.Error); ok {
				if !netErr.Temporary() {
					return nil, StoppedError
				} else if netErr.Timeout() {
					continue
				}
			}
		}

		return newConn, err
	}
}

func (sl *StoppableListener) Stop() (err error) {
	if sl.stopCh == nil {
		return
	}
	sl.log("StoppableListener stopping listening")
	if closeErr := sl.TCPListener.Close(); closeErr != nil {
		sl.log("StoppableListener non-fatal error closing underyling TCP listener: %s", closeErr)
		return
	}
	return
}

// StopSafely waits until the socket is longer reachable, or returns an error
// if the check times out.
func (sl *StoppableListener) StopSafely() (err error) {
	if err = sl.Stop(); err != nil {
		return
	}
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
	host, port, _ := net.SplitHostPort(sl.TCPListener.Addr().String())
	args := append([]string{"-w", fmt.Sprint(sl.StopCheckWaitSeconds)}, host, port)
	for i := 0; i < sl.MaxStopChecks; i++ {
		err := exec.Command("nc", args...).Run()
		if err != nil { // If `nc` exits with non-zero status code then that means the port is closed.
			sl.log("waitUntilStopped completed ok")
			return nil
		}
		sl.log("waitUntilStopped the port is still open")
		time.Sleep(time.Duration(sl.StopCheckWaitSeconds) * time.Second)
	}
	sl.log("waitUntilStopped max checks exceeded; stop failed")
	return NotStoppedError
}

func (sl *StoppableListener) log(format string, args ...interface{}) {
	if sl.Verbose {
		format = fmt.Sprintf("[bind-addr=%v] %v", sl.TCPListener.Addr().String(), format)
		log.Printf(format, args...)
	}
}
