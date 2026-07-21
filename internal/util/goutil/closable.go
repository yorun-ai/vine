package goutil

import "log/slog"

type Closable interface {
	Close() error
}

func Close(closable Closable) {
	err := closable.Close()
	if err != nil {
		slog.Warn(err.Error())
	}
}
