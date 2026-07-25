package logger

type Format string

const (
	// FormatJSON emits one JSON object per line.
	FormatJSON Format = "JSON"
	// FormatText emits one line of human-readable key=value pairs per record.
	FormatText Format = "TEXT"
)

func IsValidFormat(format Format) bool {
	return format == FormatJSON || format == FormatText
}
