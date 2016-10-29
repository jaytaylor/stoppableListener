package stoppableListener

// Many thanks to Richard Crowley for writing http://rcrowley.org/articles/golang-graceful-stop.html.

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

var (
	DefaultStopCheckTimeout  = time.Duration(1 * time.Millisecond)
	DefaultTimeoutMultiplier = 100   // Default number of timeouts permitted before giving up and returning failed-to-stop error.
	DefaultVerbose           = false // Default value for Verbose field of new StoppableListeners.

	StoppedError      = errors.New("StoppableListener: listener stopped")
	ListenerWrapError = errors.New("StoppableListener: cannot wrap listener")
	NotStoppedError   = errors.New("StoppableListener: listener failed to stop, listener port is still open after timeout")
)

type StoppableListener struct {
	*net.TCPListener                // Wrapped listener.
	stopCh            chan struct{} // Channel used only to indicate listener should shutdown.
	StopCheckTimeout  time.Duration // TCP socket timeout value used when a stop-check is run.
	TimeoutMultiplier int           // How many times the StopCheckTimeout duration should the wait-loop allow.
	Verbose           bool          // Activates verbose logging.
}

// New creates a new stoppable TCP listener.
func New(l net.Listener) (*StoppableListener, error) {
	tcpL, ok := l.(*net.TCPListener)

	if !ok {
		return nil, ListenerWrapError
	}

	sl := &StoppableListener{
		TCPListener:       tcpL,
		stopCh:            make(chan struct{}),
		StopCheckTimeout:  DefaultStopCheckTimeout,
		TimeoutMultiplier: DefaultTimeoutMultiplier,
		Verbose:           DefaultVerbose,
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
	sl.log("StoppableListener: Invoking stop-listening")
	if closeErr := sl.TCPListener.Close(); closeErr != nil {
		sl.log("StoppableListener: Non-fatal error closing underyling TCP listener: %s", closeErr)
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

// waitUntilStopped determines whether or not the listening port is still
// accepting connections.  Returns nil when connections are no longer being
// accepted, and NotStoppedError if StopCheckTimeout * TimeoutMultiplier is
// exceeded.
func (sl *StoppableListener) waitUntilStopped() error {
	var (
		waitUntil = time.Now().Add(time.Duration(sl.TimeoutMultiplier) * sl.StopCheckTimeout)
		addr      = sl.TCPListener.Addr().String()
	)
	for i := 0; time.Now().Before(waitUntil); i++ {
		conn, err := net.DialTimeout("tcp", addr, sl.StopCheckTimeout)
		if err != nil {
			sl.log("StoppableListener: Dial error=%s (waitUntilStop done!)", err)
			return nil
		}
		conn.Close()
		time.Sleep(10 * time.Millisecond)
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
