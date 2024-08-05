package logwriter

import "bytes"

type LogFunc func(string)

// LogWriter is a writer that simply logs the output using the given logFunc
type LogWriter struct {
	logFunc LogFunc
	prefix  string
	suffix  string
}

// NewLogWriter creates a new LogWriter
func NewLogWriter(prefix, suffix string, logFunc LogFunc) LogWriter {
	return LogWriter{
		logFunc: logFunc,
		prefix:  prefix,
		suffix:  suffix,
	}
}

// Write logs the given bytes using the logFunc
func (w LogWriter) Write(p []byte) (n int, err error) {
	lines := bytes.Split(p, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		newLine := string(line) // psw.prefix + string(line) + psw.suffix
		w.logFunc(newLine)
	}

	return len(p), nil
}
