package embedded

import (
	"errors"
	"time"

	"github.com/gobwas/glob"
	"go.yorun.ai/vine/util/vslice"
)

const scanDefaultCount = 10
const scanCursorTTL = time.Minute

type _ScanOption struct {
	cursor uint64
	match  string
	count  int
}

type _ScanCursor struct {
	keys     []string
	index    int
	expireAt time.Time
}

func (s *Store) ScanKeys(cursor uint64, match string, count int) ([]string, uint64, error) {
	return s.scanKeys(_ScanOption{
		cursor: cursor,
		match:  match,
		count:  count,
	})
}

func (s *Store) scanKeys(option _ScanOption) ([]string, uint64, error) {
	now := timeNow()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.cleanExpiredScanCursorsLocked(now)
	if option.cursor != 0 {
		return s.scanCursorKeysLocked(option, now)
	}

	keys, err := s.scanAllKeysLocked(option, now)
	if err != nil {
		return nil, 0, err
	}
	if len(keys) <= option.count {
		return keys, 0, nil
	}

	cursor := s.newScanCursorLocked(keys[option.count:], now)
	return keys[:option.count], cursor, nil
}

func (s *Store) scanAllKeys(option _ScanOption) ([]string, error) {
	now := timeNow()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cleanExpiredScanCursorsLocked(now)
	return s.scanAllKeysLocked(option, now)
}

func (s *Store) scanAllKeysLocked(option _ScanOption, now time.Time) ([]string, error) {
	pattern, err := glob.Compile(option.match)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0)
	for key, item := range s.items {
		if item.expired(now) {
			delete(s.items, key)
			continue
		}
		if !pattern.Match(key) {
			continue
		}
		keys = append(keys, key)
	}
	return vslice.Sort(keys), nil
}

func (s *Store) scanCursorKeysLocked(option _ScanOption, now time.Time) ([]string, uint64, error) {
	cursor, ok := s.scans[option.cursor]
	if !ok {
		return nil, 0, errors.New("invalid scan cursor")
	}

	end := option.count
	if end > len(cursor.keys) {
		end = len(cursor.keys)
	}
	keys := vslice.Clone(cursor.keys[:end])
	cursor.keys = cursor.keys[end:]
	if len(cursor.keys) == 0 {
		delete(s.scans, option.cursor)
		return keys, 0, nil
	}

	cursor.index += end
	cursor.expireAt = now.Add(scanCursorTTL)
	s.scans[option.cursor] = cursor
	return keys, option.cursor, nil
}

func (s *Store) newScanCursorLocked(keys []string, now time.Time) uint64 {
	s.nextScan++
	if s.nextScan == 0 {
		s.nextScan++
	}
	cursor := s.nextScan
	s.scans[cursor] = _ScanCursor{
		keys:     vslice.Clone(keys),
		expireAt: now.Add(scanCursorTTL),
	}
	return cursor
}

func (s *Store) cleanExpiredScanCursorsLocked(now time.Time) {
	for cursor, scan := range s.scans {
		if !now.Before(scan.expireAt) {
			delete(s.scans, cursor)
		}
	}
}
