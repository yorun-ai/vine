package rdb

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/vine/infra/rdb/adapter"
	"gorm.io/gorm"
)

const (
	defaultMaxOpenConns = 10

	connMaxIdleTime = 10 * time.Minute
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

	gormDB, err := gorm.Open(adapter.NewDialector(config.ConnURL), &gorm.Config{
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

func configurePool(gormDB *gorm.DB, config Option) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	maxOpenConns := defaultMaxOpenConns
	if config.MaxOpenConn > 0 {
		maxOpenConns = config.MaxOpenConn
	}
	sqlDB.SetMaxIdleConns(maxOpenConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	sqlDB.SetConnMaxLifetime(connMaxLifeTime)
	if dsn, ok := strings.CutPrefix(config.ConnURL, "sqlite://"); ok {
		_, query, _ := strings.Cut(dsn, "?")
		params, _ := url.ParseQuery(query)
		if dsn == ":memory:" || params.Get("mode") == "memory" {
			// An in-memory database disappears when its last connection closes.
			sqlDB.SetConnMaxIdleTime(0)
			sqlDB.SetConnMaxLifetime(0)
		}
	}
	return nil
}
