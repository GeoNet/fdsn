package sl

import (
	"context"
	"fmt"
	"net"
	"time"
)

// SLink is a wrapper around an SLConn to provide
// handling of timeouts and keep alive messages.
type SLink struct {
	Server  string
	Timeout time.Duration

	NetTo     time.Duration
	KeepAlive time.Duration
	Strict    bool

	Start    time.Time
	End      time.Time
	Sequence int

	Streams   string
	Selectors string

	State []Station
}

// SLinkOpt is a function for setting SLink internal parameters.
type SLinkOpt func(*SLink)

// SetServer sets the seedlink server in the form of "host<:port>".
func SetServer(v string) SLinkOpt {
	return func(s *SLink) {
		s.Server = v
	}
}

// SetTimeout sets the timeout for seedlink server commands and packet requests.
func SetTimeout(d time.Duration) SLinkOpt {
	return func(s *SLink) {
		s.Timeout = d
	}
}

// SetNetTo sets the time to after which the connection is closed after no packets have been received.
func SetNetTo(d time.Duration) SLinkOpt {
	return func(s *SLink) {
		s.NetTo = d
	}
}

// SetKeepAlive sets the time to send an ID message to server if no packets have been received.
func SetKeepAlive(d time.Duration) SLinkOpt {
	return func(s *SLink) {
		s.KeepAlive = d
	}
}

// SetSequence sets the start sequence for the initial request.
func SetSequence(sequence int) SLinkOpt {
	return func(s *SLink) {
		s.Sequence = sequence
	}
}

// SetStart sets the start of the initial request from the seedlink server.
func SetStart(t time.Time) SLinkOpt {
	return func(s *SLink) {
		s.Start = t.UTC()
	}
}

// SetEndTime sets the end of the initial request from the seedlink server.
func SetEnd(t time.Time) SLinkOpt {
	return func(s *SLink) {
		s.End = t.UTC()
	}
}

// SetStreams sets the list of stations and streams to from the seedlink server.
func SetStreams(streams string) SLinkOpt {
	return func(s *SLink) {
		s.Streams = streams
	}
}

// SetSelectors sets the default list of selectors to use for seedlink stream requests.
func SetSelectors(selectors string) SLinkOpt {
	return func(s *SLink) {
		s.Selectors = selectors
	}
}

// SetState sets the default list of station state information, only used during the initial connection.
func SetState(stations ...Station) SLinkOpt {
	return func(s *SLink) {
		s.State = append(s.State, stations...)
	}
}

// SetStrict sets whether a package error should restart the collection system, rather than be skipped.
func SetStrict(strict bool) SLinkOpt {
	return func(s *SLink) {
		s.Strict = strict
	}
}

// NewSlink returns a SLink pointer for the given server, optional settings can be passed as SLinkOpt functions.
func NewSLink(opts ...SLinkOpt) *SLink {
	sl := SLink{
		Server:    "localhost:18000",
		Streams:   "*_*",
		Selectors: "???",
		Timeout:   5 * time.Second,
		NetTo:     300 * time.Second,
		KeepAlive: 30 * time.Second,
		Sequence:  -1,
	}
	for _, opt := range opts {
		opt(&sl)
	}
	return &sl
}

// SetTimeout sets the timeout value used for connection requests.
func (s *SLink) SetTimeout(d time.Duration) {
	s.Timeout = d
}

// SetNetTo sets the overall timeout after which a reconnection is tried.
func (s *SLink) SetNetTo(d time.Duration) {
	s.NetTo = d
}

// SetKeepAlive sets the time interval needed without any packets for
// a check message is sent.
func (s *SLink) SetKeepAlive(d time.Duration) {
	s.KeepAlive = d
}

// SetSequence sets the start sequence for the initial request.
func (s *SLink) SetSequence(sequence int) {
	s.Sequence = sequence
}

// SetStartTime sets the initial starting time of the request.
func (s *SLink) SetStart(t time.Time) {
	s.Start = t.UTC()
}

// SetEndTime sets the initial end time of the request.
func (s *SLink) SetEnd(t time.Time) {
	s.End = t.UTC()
}

// SetStreams sets the channel streams used for seedlink connections.
func (s *SLink) SetStreams(streams string) {
	s.Streams = streams
}

// SetSelectors sets the channel selectors used for seedlink connections.
func (s *SLink) SetSelectors(selectors string) {
	s.Selectors = selectors
}

// SetState sets the default list of station state information.
func (s *SLink) SetState(stations ...Station) {
	s.State = append([]Station{}, stations...)
}

// AddState appends the list of station state information.
func (s *SLink) AddState(stations ...Station) {
	s.State = append(s.State, stations...)
}

// NewSlink returns a SLink pointer for the given server, optional settings can be passed

// CollectFunc is a function run on each returned seedlink packet. It should return a true value
// to stop collecting data without an error message. A non-nil returned error will also stop
// collection but with an assumed errored state.
type CollectFunc func(string, []byte) (bool, error)

// CollectWithContext makes a connection to the seedlink server, recovers initial client information and
// the sets the connection into streaming mode. Recovered packets are passed to a given function
// to process, if this function returns a true value or a non-nil error value the collection will
// stop and the function will return.
// If a call returns with a timeout error a check is made whether a keepalive is needed or whether
// the function should return as no data has been received for an extended period of time. It is
// assumed the calling function will attempt a reconnection with an updated set of options, specifically
// any start or end time parameters. The Context parameter can be used to to cancel the data collection
// independent of the function as this may never be called if no appropriate has been received.
func (s *SLink) CollectWithContext(ctx context.Context, fn CollectFunc) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var state State
	for _, v := range s.State {
		state.Add(v)
	}

	list, err := decodeStreams(s.Streams, s.Selectors)
	if err != nil {
		return err
	}

	conn, err := NewConn(s.Server, s.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, l := range list {
		if err := conn.CommandStation(l.station, l.network); err != nil {
			return err
		}

		if err := conn.CommandSelect(l.selection); err != nil {
			return err
		}

		sequence, starttime := s.Sequence, s.Start
		if v, ok := state.Find(Station{Network: l.network, Station: l.station}); ok {
			sequence, starttime = v.Sequence, v.Timestamp
		}

		switch {
		case !s.End.IsZero():
			if err := conn.CommandTime(s.Start, s.End); err != nil {
				return err
			}
			// there may be a sequence number
		case !(sequence < 0):
			//convert the next sequence number into uppercase hex
			seq := fmt.Sprintf("%06X", (s.Sequence+1)&0xffffff)
			if err := conn.CommandData(seq, starttime); err != nil {
				return err
			}
		default:
			// or check a possible start time
			if err := conn.CommandTime(starttime, time.Time{}); err != nil {
				return err
			}
		}
	}
	if err := conn.CommandEnd(); err != nil {
		return err
	}

	last := time.Now()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			switch pkt, err := conn.Collect(); {
			case err != nil:
				switch err := err.(type) {
				case net.Error:
					switch {
					case err.Timeout():
						// hit the limit so close the connection
						if s.NetTo > 0 && s.NetTo < time.Since(last) {
							return err
						}
						// may be time for a keep alive
						if s.KeepAlive > 0 && s.KeepAlive < time.Since(last) {
							// send an ID request, ignore any results other than an error
							if _, err := conn.CommandId(); err != nil {
								return err
							}
							last = time.Now()
						}
					default:
						// not a timeout
						return err
					}
				case *PacketError:
					if s.Strict {
						return err
					}
				default:
					return err
				}
			case pkt != nil:
				if stop, err := fn(string(pkt.Seq[:]), pkt.Data[:]); err != nil || stop {
					return err
				}
				last = time.Now()
			}
		}
	}

	return nil
}

// Collect calls CollectWithContext with a background Context and a handler function.
func (s *SLink) Collect(fn CollectFunc) error {
	return s.CollectWithContext(context.Background(), fn)
}
