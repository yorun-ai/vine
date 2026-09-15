// Package lock provides automatically renewed distributed lease locks through Link.
// Obtain *Locker or *UniversalLocker through application dependency injection.
// Locker isolates keys by application name. UniversalLocker shares keys across
// applications connected to the same lock backend, including across Hubs using
// the same external Redis database. The two scopes have separate key prefixes.
//
// Lock waits for ownership; TryLock attempts acquisition once. Both return leases
// that must be released with Unlock or TryUnlock. Contention is not an error;
// backend failures and cancelled or timed-out acquisition panic with framework errors.
//
// WithTTL configures the lease duration, not the wait limit. Use Locker.WithContext
// with a deadline to bound waiting and the lifetime of the acquired lease. Work must
// observe Lock.Context: cancellation, expiry and renewal failure invalidate ownership.
// Successful release also cancels that context but does not mark the lease broken.
// These locks do not provide fairness, reentrancy or fencing.
package lock
