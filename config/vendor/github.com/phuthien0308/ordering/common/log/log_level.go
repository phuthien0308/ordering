package log

//go:generate stringer -type=LogLevel
type LogLevel uint

const (
	DEBUG LogLevel = iota
	INFO
	WARM
	ERROR
	FALTAL
)

var LogLevelString = map[LogLevel]string{
	DEBUG:  "DEBUG",
	INFO:   "INFO",
	WARM:   "WARM",
	ERROR:  "ERROR",
	FALTAL: "FALTALs",
}

func (l LogLevel) String() string {
	if v, ok := LogLevelString[l]; ok {
		return v
	}
	return ""
}
