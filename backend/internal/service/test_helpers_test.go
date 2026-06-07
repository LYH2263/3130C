package service

import (
	"log/slog"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"label3130/backend/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	err = db.AutoMigrate(
		&models.ClassRoom{},
		&models.User{},
		&models.Question{},
		&models.QuestionOption{},
		&models.Attempt{},
		&models.AttemptAnswer{},
	)
	if err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Discard, nil))
}

func seedQuestions(t *testing.T, db *gorm.DB, questions []models.Question) {
	t.Helper()
	for i := range questions {
		if err := db.Create(&questions[i]).Error; err != nil {
			t.Fatalf("failed to seed question %d: %v", i, err)
		}
	}
}
