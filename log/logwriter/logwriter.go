package logwriter

type LogFunc func(string)

// LogWriter is a writer that simply logs the output using the given logFunc
type LogWriter struct {
	logFunc LogFunc
}

// NewLogWriter creates a new LogWriter
func NewLogWriter(logFunc LogFunc) LogWriter {
	return LogWriter{
		logFunc: logFunc,
	}
}

// Write logs the given bytes using the logFunc
func (w LogWriter) Write(p []byte) (n int, err error) {
	w.logFunc(string(p))
	return len(p), nil
}
