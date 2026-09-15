package lock

import (
	"time"

	internallock "go.yorun.ai/vine/internal/core/lock"
)

// Locker acquires locks shared by instances of the same application.
type Locker = internallock.Locker

// UniversalLocker acquires locks shared across applications using the same lock backend.
type UniversalLocker = internallock.UniversalLocker

// Lock represents an acquired lease.
type Lock = internallock.Lock

// OptionFunc configures a lock acquisition.
type OptionFunc = internallock.OptionFunc

// WithTTL sets the automatically renewed lease duration.
func WithTTL(duration time.Duration) OptionFunc {
	return internallock.WithTTL(duration)
}
