package metrics

import "context"

// Message defines an interface for processing bytes.
type Processor interface {
	Process([]byte) error
}

// processor with passed in context to catch possible system stop signal
type ProcessorWithContext interface {
	Process(context.Context, []byte) error
}

// DoProcess executes m.Process with messaging metrics.
func DoProcess(m Processor, b []byte) error {
	MsgRx()
	t := Start()
	defer func() {
		_ = t.Track("process") // we keep this package simple thus not reporting this error
	}()

	err := m.Process(b)

	switch err {
	case nil:
		MsgProc()
	default:
		MsgErr()
	}

	return err
}

// DoProcess executes m.Process with messaging metrics.
// pass a context to processor to catch system stop signal
func DoProcessWithContext(ctx context.Context, m ProcessorWithContext, b []byte) error {
	MsgRx()
	t := Start()
	defer func() {
		_ = t.Track("process") // we keep this package simple thus not reporting this error
	}()

	err := m.Process(ctx, b)

	switch err {
	case nil:
		MsgProc()
	default:
		MsgErr()
	}

	return err
}

// Logger defines an interface for logging.
type Logger interface {
	Printf(string, ...interface{})
}

type discarder struct {
}

func (d discarder) Printf(string, ...interface{}) {
}
