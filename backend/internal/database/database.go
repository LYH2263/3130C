package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"label3130/backend/internal/models"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Connect(dsn string, poolCfg PoolConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(poolCfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolCfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)

	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("resolve sql db: %w", err)
	}
	return sqlDB.Close()
}

func autoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.ClassRoom{},
		&models.User{},
		&models.Question{},
		&models.QuestionOption{},
		&models.Attempt{},
		&models.AttemptAnswer{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
