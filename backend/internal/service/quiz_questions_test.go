package service

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"gorm.io/gorm"

	"label3130/backend/internal/models"
)

func TestQuestionService_GetQuizQuestions(t *testing.T) {
	t.Parallel()

	t.Run("limit behavior", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name          string
			questionCount int
			limit         int
			expectedCount int
			expectErr     bool
		}{
			{
				name:          "limit 0 returns all",
				questionCount: 5,
				limit:         0,
				expectedCount: 5,
				expectErr:     false,
			},
			{
				name:          "limit less than total",
				questionCount: 5,
				limit:         3,
				expectedCount: 3,
				expectErr:     false,
			},
			{
				name:          "limit equals total",
				questionCount: 3,
				limit:         3,
				expectedCount: 3,
				expectErr:     false,
			},
			{
				name:          "limit greater than total returns all",
				questionCount: 3,
				limit:         10,
				expectedCount: 3,
				expectErr:     false,
			},
			{
				name:          "limit 1",
				questionCount: 5,
				limit:         1,
				expectedCount: 1,
				expectErr:     false,
			},
			{
				name:          "negative limit returns all",
				questionCount: 4,
				limit:         -1,
				expectedCount: 4,
				expectErr:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				db := setupTestDB(t)
				seedNQuestions(t, db, tt.questionCount)
				svc := NewQuestionService(db, testLogger())

				questions, err := svc.GetQuizQuestions(tt.limit)

				if tt.expectErr {
					if err == nil {
						t.Fatalf("GetQuizQuestions() expected error, got nil; questions=%d", len(questions))
					}
					return
				}

				if err != nil {
					t.Fatalf("GetQuizQuestions() unexpected error: %v", err)
				}
				if len(questions) != tt.expectedCount {
					t.Errorf("GetQuizQuestions() returned %d questions, want %d", len(questions), tt.expectedCount)
				}
			})
		}
	})

	t.Run("empty question bank returns error", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		svc := NewQuestionService(db, testLogger())

		questions, err := svc.GetQuizQuestions(5)

		if err == nil {
			t.Fatalf("GetQuizQuestions() expected error for empty bank, got nil; questions=%+v", questions)
		}
		if !errors.Is(err, ErrNoQuestions) {
			t.Fatalf("GetQuizQuestions() error = %v, want ErrNoQuestions", err)
		}
	})

	t.Run("question set is consistent - same IDs returned in order", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		seedNQuestions(t, db, 5)
		svc := NewQuestionService(db, testLogger())

		result1, err := svc.GetQuizQuestions(0)
		if err != nil {
			t.Fatalf("first call unexpected error: %v", err)
		}

		result2, err := svc.GetQuizQuestions(0)
		if err != nil {
			t.Fatalf("second call unexpected error: %v", err)
		}

		if len(result1) != len(result2) {
			t.Fatalf("result length mismatch: %d vs %d", len(result1), len(result2))
		}

		for i := range result1 {
			if result1[i].ID != result2[i].ID {
				t.Errorf("question order changed at index %d: id=%d vs id=%d", i, result1[i].ID, result2[i].ID)
			}
			if result1[i].Title != result2[i].Title {
				t.Errorf("question title mismatch at index %d: %q vs %q", i, result1[i].Title, result2[i].Title)
			}
		}
	})

	t.Run("options are shuffled but option set is unchanged", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		seedNQuestions(t, db, 3)
		svc := NewQuestionService(db, testLogger())

		result, err := svc.GetQuizQuestions(0)
		if err != nil {
			t.Fatalf("GetQuizQuestions() unexpected error: %v", err)
		}

		for qi, q := range result {
			if len(q.Options) < 2 {
				t.Fatalf("question %d has only %d options, expected at least 2", qi, len(q.Options))
			}

			var origOptions []models.QuestionOption
			if err := db.Where("question_id = ?", q.ID).Order("id asc").Find(&origOptions).Error; err != nil {
				t.Fatalf("failed to load original options: %v", err)
			}

			if len(q.Options) != len(origOptions) {
				t.Errorf("question %d: returned %d options, db has %d", qi, len(q.Options), len(origOptions))
				continue
			}

			returnedIDs := make([]uint, len(q.Options))
			for i, opt := range q.Options {
				returnedIDs[i] = opt.ID
			}

			origIDs := make([]uint, len(origOptions))
			for i, opt := range origOptions {
				origIDs[i] = opt.ID
			}

			sortedReturned := make([]uint, len(returnedIDs))
			copy(sortedReturned, returnedIDs)
			sort.Slice(sortedReturned, func(i, j int) bool { return sortedReturned[i] < sortedReturned[j] })

			sortedOrig := make([]uint, len(origIDs))
			copy(sortedOrig, origIDs)
			sort.Slice(sortedOrig, func(i, j int) bool { return sortedOrig[i] < sortedOrig[j] })

			for i := range sortedReturned {
				if sortedReturned[i] != sortedOrig[i] {
					t.Errorf("question %d: option set mismatch at position %d: %d vs %d", qi, i, sortedReturned[i], sortedOrig[i])
				}
			}

			optContentMap := make(map[uint]string)
			for _, opt := range origOptions {
				optContentMap[opt.ID] = opt.Content
			}
			for _, opt := range q.Options {
				if content, ok := optContentMap[opt.ID]; !ok {
					t.Errorf("question %d: unknown option id %d", qi, opt.ID)
				} else if opt.Content != content {
					t.Errorf("question %d: option %d content mismatch: %q vs %q", qi, opt.ID, opt.Content, content)
				}
			}
		}
	})

	t.Run("shuffling produces different order across calls (probabilistic)", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		seedNQuestions(t, db, 1)
		svc := NewQuestionService(db, testLogger())

		const trials = 20
		allSameOrder := true
		var firstOrder []uint

		for i := 0; i < trials; i++ {
			result, err := svc.GetQuizQuestions(0)
			if err != nil {
				t.Fatalf("trial %d: unexpected error: %v", i, err)
			}
			if len(result) != 1 {
				t.Fatalf("trial %d: expected 1 question, got %d", i, len(result))
			}

			optIDs := make([]uint, len(result[0].Options))
			for j, opt := range result[0].Options {
				optIDs[j] = opt.ID
			}

			if firstOrder == nil {
				firstOrder = optIDs
				continue
			}

			same := true
			for k := range firstOrder {
				if firstOrder[k] != optIDs[k] {
					same = false
					break
				}
			}
			if !same {
				allSameOrder = false
				break
			}
		}

		if allSameOrder {
			t.Log("WARNING: options were in same order across all trials (possible but unlikely)")
		}
	})
}

func seedNQuestions(t *testing.T, db *gorm.DB, n int) {
	t.Helper()

	for i := 0; i < n; i++ {
		q := models.Question{
			Title:       fmt.Sprintf("Question %d", i+1),
			Description: fmt.Sprintf("Description for question %d", i+1),
			CreatedBy:   1,
			Options: []models.QuestionOption{
				{Content: "Option A", IsCorrect: true},
				{Content: "Option B", IsCorrect: false},
				{Content: "Option C", IsCorrect: false},
				{Content: "Option D", IsCorrect: false},
			},
		}
		if err := db.Create(&q).Error; err != nil {
			t.Fatalf("failed to create question %d: %v", i, err)
		}
	}
}
