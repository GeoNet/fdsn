package metrics

// Message defines an interface for processing bytes.
type Processor interface {
	Process([]byte) error
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

// Logger defines an interface for logging.
type Logger interface {
	Printf(string, ...interface{})
}

type discarder struct {
}

func (d discarder) Printf(string, ...interface{}) {
}
