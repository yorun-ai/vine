package rdb

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultMaxOpenConns = 10
	maxIdleConnsPercent = 0.3

	connMaxIdleTime = time.Hour * 1
	connMaxLifeTime = time.Hour * 8
)

var (
	sharedGormDBsMu sync.Mutex
	sharedGormDBs   = map[string]*_SharedGormDB{}
)

type _SharedGormDB struct {
	gormDB   *gorm.DB
	refCount int
}

func openConnection(config Option) (*gorm.DB, error) {
	if config.ConnURL == "" {
		return nil, fmt.Errorf("rdb connUrl is empty")
	}

	sharedGormDBsMu.Lock()
	if shared, ok := sharedGormDBs[config.ConnURL]; ok {
		shared.refCount++
		gormDB := shared.gormDB
		sharedGormDBsMu.Unlock()
		return gormDB, nil
	}

	gormDB, err := gorm.Open(newDialector(config.ConnURL), &gorm.Config{
		Logger: newLogger(),
	})
	if err != nil {
		sharedGormDBsMu.Unlock()
		return nil, err
	}

	err = configurePool(gormDB, config)
	if err != nil {
		sharedGormDBsMu.Unlock()
		return nil, err
	}

	sharedGormDBs[config.ConnURL] = &_SharedGormDB{
		gormDB:   gormDB,
		refCount: 1,
	}
	sharedGormDBsMu.Unlock()
	return gormDB, nil
}

func closeConnection(connURL string) {
	sharedGormDBsMu.Lock()
	shared, ok := sharedGormDBs[connURL]
	if !ok {
		sharedGormDBsMu.Unlock()
		return
	}

	shared.refCount--
	if shared.refCount > 0 {
		sharedGormDBsMu.Unlock()
		return
	}

	delete(sharedGormDBs, connURL)
	sharedGormDBsMu.Unlock()

	sqlDB, err := shared.gormDB.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func newDialector(connURL string) gorm.Dialector {
	switch {
	case strings.HasPrefix(connURL, "sqlite://"):
		return sqlite.Open(strings.TrimPrefix(connURL, "sqlite://"))
	default:
		return postgres.Open(connURL)
	}
}

func configurePool(gormDB *gorm.DB, config Option) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	maxOpenConns := defaultMaxOpenConns
	if config.MaxOpenConn > 0 {
		maxOpenConns = config.MaxOpenConn
	}
	maxIdleConns := int(math.Ceil(float64(maxOpenConns) * maxIdleConnsPercent))

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	sqlDB.SetConnMaxLifetime(connMaxLifeTime)
	return nil
}
