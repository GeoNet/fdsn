package sl

import (
	"context"
	"os"
	"time"
)

// SLConn is a wrapper around SLink to manage state.
type SLConn struct {
	*SLink

	StateFile string
	Flush     time.Duration
	Delay     time.Duration
}

// SLinkOpt is a function for setting SLink internal parameters.
type SLConnOpt func(*SLConn)

// SetStateFile sets the connection state file.
func SetStateFile(v string) SLConnOpt {
	return func(s *SLConn) {
		s.StateFile = v
		s.Flush = 300 * time.Second
		s.Delay = 30 * time.Second
	}
}

// SetFlush sets how often the state file should be flushed
func SetFlush(v time.Duration) SLConnOpt {
	return func(s *SLConn) {
		s.Flush = v
	}
}

// SetDelay sets how long to wait until retrying a network connection
func SetDelay(v time.Duration) SLConnOpt {
	return func(s *SLConn) {
		s.Delay = v
	}
}

// NewSLConn builds a SLConn from a SLink and any extra options.
func NewSLConn(slink *SLink, opts ...SLConnOpt) *SLConn {
	slconn := SLConn{
		SLink: slink,
	}
	for _, opt := range opts {
		opt(&slconn)
	}
	return &slconn
}

func (s *SLConn) CollectWithContext(ctx context.Context, fn CollectFunc) error {

	// keep track
	var state State

	// possibly need to load the state file
	if s.StateFile != "" {
		info, err := os.Stat(s.StateFile)
		if !os.IsNotExist(err) && !info.IsDir() {
			if err := state.ReadFile(s.StateFile); err != nil {
				return err
			}
		}
	}

	// keep writing the state files
	if s.Flush > 0 && s.StateFile != "" {
		go func() {
			for range time.Tick(s.Flush) {
				_ = state.WriteFile(s.StateFile)
			}
		}()
		// and after an uneventful end
		defer func() {
			_ = state.WriteFile(s.StateFile)
		}()
	}

loop:
	for {
		// add any current state to a new connection
		s.AddState(state.Stations()...)

		// keep connection open ...
		if err := s.SLink.CollectWithContext(ctx, func(seq string, data []byte) (bool, error) {
			if ok, err := fn(seq, data); ok || err != nil {
				return ok, err
			}
			// manage state
			state.Add(UnpackStation(seq, data))
			return false, nil
		}); err != nil {
			return err
		}

		// may want to wait before a retry
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(s.Delay):
		}

	}

	// loop setup connection
	return nil
}

// Collect calls CollectWithContext with a background Context and a handler function.
func (s *SLConn) Collect(fn CollectFunc) error {
	return s.CollectWithContext(context.Background(), fn)
}
