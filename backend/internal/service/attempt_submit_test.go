package service

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

func TestAttemptService_Submit(t *testing.T) {
	t.Parallel()

	t.Run("score and rate calculation scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name          string
			answers       []dto.SubmitAnswerItem
			expectedScore int
			expectedTotal int
			expectedRate  string
		}{
			{
				name:          "all correct",
				answers:       []dto.SubmitAnswerItem{{QuestionID: 1, OptionID: 1}},
				expectedScore: 1,
				expectedTotal: 1,
				expectedRate:  "100%",
			},
			{
				name: "partially correct - 2 of 3",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 1},
					{QuestionID: 2, OptionID: 3},
					{QuestionID: 3, OptionID: 6},
				},
				expectedScore: 2,
				expectedTotal: 3,
				expectedRate:  "67%",
			},
			{
				name: "all wrong",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 2},
					{QuestionID: 2, OptionID: 4},
				},
				expectedScore: 0,
				expectedTotal: 2,
				expectedRate:  "0%",
			},
			{
				name: "single question wrong",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 3, OptionID: 7},
				},
				expectedScore: 0,
				expectedTotal: 1,
				expectedRate:  "0%",
			},
			{
				name: "50 percent rate",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 1},
					{QuestionID: 2, OptionID: 4},
				},
				expectedScore: 1,
				expectedTotal: 2,
				expectedRate:  "50%",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				db := setupTestDB(t)
				seedThreeQuestions(t, db)
				svc := NewAttemptService(db, testLogger())

				result, err := svc.Submit(1, 1, dto.SubmitRequest{Answers: tt.answers})

				if err != nil {
					t.Fatalf("Submit() unexpected error: %v", err)
				}
				if result.Score != tt.expectedScore {
					t.Errorf("Submit() score = %d, want %d", result.Score, tt.expectedScore)
				}
				if result.Total != tt.expectedTotal {
					t.Errorf("Submit() total = %d, want %d", result.Total, tt.expectedTotal)
				}
				if result.Rate != tt.expectedRate {
					t.Errorf("Submit() rate = %q, want %q", result.Rate, tt.expectedRate)
				}
				if result.AttemptID == 0 {
					t.Error("Submit() attemptID should not be zero")
				}
			})
		}
	})

	t.Run("error scenarios", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			answers     []dto.SubmitAnswerItem
			expectErr   error
			errContains string
		}{
			{
				name:        "empty answers",
				answers:     []dto.SubmitAnswerItem{},
				expectErr:   ErrInvalidSubmission,
				errContains: "",
			},
			{
				name:        "nil answers",
				answers:     nil,
				expectErr:   ErrInvalidSubmission,
				errContains: "",
			},
			{
				name: "non-existent questionId",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 9999, OptionID: 1},
				},
				expectErr:   ErrNoQuestions,
				errContains: "",
			},
			{
				name: "one valid one invalid questionId",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 1},
					{QuestionID: 9999, OptionID: 1},
				},
				expectErr:   ErrInvalidSubmission,
				errContains: "",
			},
			{
				name: "non-existent optionId",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 9999},
				},
				expectErr:   ErrInvalidSubmission,
				errContains: "",
			},
			{
				name: "option belongs to different question",
				answers: []dto.SubmitAnswerItem{
					{QuestionID: 1, OptionID: 3},
				},
				expectErr:   ErrInvalidSubmission,
				errContains: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				db := setupTestDB(t)
				seedThreeQuestions(t, db)
				svc := NewAttemptService(db, testLogger())

				result, err := svc.Submit(1, 1, dto.SubmitRequest{Answers: tt.answers})

				if err == nil {
					t.Fatalf("Submit() expected error, got nil; result=%+v", result)
				}
				if tt.expectErr != nil {
					if !errors.Is(err, tt.expectErr) && !strings.Contains(err.Error(), tt.expectErr.Error()) {
						t.Fatalf("Submit() error = %v, want error %v", err, tt.expectErr)
					}
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("Submit() error = %v, want error containing %q", err, tt.errContains)
				}
			})
		}
	})

	t.Run("duplicate questionId submission - known behavior", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		seedThreeQuestions(t, db)
		svc := NewAttemptService(db, testLogger())

		answers := []dto.SubmitAnswerItem{
			{QuestionID: 1, OptionID: 1},
			{QuestionID: 1, OptionID: 1},
			{QuestionID: 1, OptionID: 2},
		}

		result, err := svc.Submit(1, 1, dto.SubmitRequest{Answers: answers})

		if err != nil {
			t.Fatalf("Submit() unexpected error: %v", err)
		}

		if result.Total != 3 {
			t.Errorf("Submit() total = %d, want 3 (duplicate questionId counted multiple times in total)", result.Total)
		}
		if result.Score != 2 {
			t.Errorf("Submit() score = %d, want 2 (each duplicate answer scored independently)", result.Score)
		}
		expectedRate := "67%"
		if result.Rate != expectedRate {
			t.Errorf("Submit() rate = %q, want %q", result.Rate, expectedRate)
		}

		var attemptCount int64
		db.Model(&models.Attempt{}).Count(&attemptCount)
		if attemptCount != 1 {
			t.Errorf("expected 1 attempt record, got %d", attemptCount)
		}

		var answerCount int64
		db.Model(&models.AttemptAnswer{}).Count(&answerCount)
		if answerCount != 3 {
			t.Errorf("expected 3 answer records (duplicates preserved), got %d", answerCount)
		}
	})

	t.Run("attempt is persisted to database", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		seedThreeQuestions(t, db)
		svc := NewAttemptService(db, testLogger())

		answers := []dto.SubmitAnswerItem{
			{QuestionID: 1, OptionID: 1},
			{QuestionID: 2, OptionID: 3},
		}

		result, err := svc.Submit(42, 10, dto.SubmitRequest{Answers: answers})
		if err != nil {
			t.Fatalf("Submit() unexpected error: %v", err)
		}

		var attempt models.Attempt
		if err := db.Preload("Answers").First(&attempt, result.AttemptID).Error; err != nil {
			t.Fatalf("failed to load attempt: %v", err)
		}

		if attempt.UserID != 42 {
			t.Errorf("attempt.UserID = %d, want 42", attempt.UserID)
		}
		if attempt.ClassID != 10 {
			t.Errorf("attempt.ClassID = %d, want 10", attempt.ClassID)
		}
		if attempt.Score != 2 {
			t.Errorf("attempt.Score = %d, want 2", attempt.Score)
		}
		if attempt.Total != 2 {
			t.Errorf("attempt.Total = %d, want 2", attempt.Total)
		}
		if len(attempt.Answers) != 2 {
			t.Errorf("len(attempt.Answers) = %d, want 2", len(attempt.Answers))
		}
	})
}

func seedThreeQuestions(t *testing.T, db *gorm.DB) {
	t.Helper()

	questions := []models.Question{
		{
			Title:       "Q1",
			Description: "First question",
			CreatedBy:   1,
			Options: []models.QuestionOption{
				{Content: "Correct A", IsCorrect: true},
				{Content: "Wrong B", IsCorrect: false},
			},
		},
		{
			Title:       "Q2",
			Description: "Second question",
			CreatedBy:   1,
			Options: []models.QuestionOption{
				{Content: "Wrong A", IsCorrect: false},
				{Content: "Correct B", IsCorrect: true},
				{Content: "Wrong C", IsCorrect: false},
			},
		},
		{
			Title:       "Q3",
			Description: "Third question",
			CreatedBy:   1,
			Options: []models.QuestionOption{
				{Content: "Wrong A", IsCorrect: false},
				{Content: "Wrong B", IsCorrect: false},
				{Content: "Correct C", IsCorrect: true},
				{Content: "Wrong D", IsCorrect: false},
			},
		},
	}

	for i := range questions {
		if err := db.Create(&questions[i]).Error; err != nil {
			t.Fatalf("failed to seed question %d: %v", i, err)
		}
	}
}
