package service

import (
	"strings"
	"testing"

	"label3130/backend/internal/models"
)

func TestQuestionService_UploadFromJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		payload       []byte
		createdBy     uint
		expectedCount int
		expectErr     bool
		errContains   string
	}{
		{
			name: "top-level object with questions array",
			payload: []byte(`{
				"questions": [
					{
						"title": "Question 1",
						"description": "Desc 1",
						"options": [
							{"content": "A", "isCorrect": true},
							{"content": "B", "isCorrect": false}
						]
					},
					{
						"title": "Question 2",
						"description": "Desc 2",
						"options": [
							{"content": "X", "isCorrect": false},
							{"content": "Y", "isCorrect": true},
							{"content": "Z", "isCorrect": false}
						]
					}
				]
			}`),
			createdBy:     1,
			expectedCount: 2,
			expectErr:     false,
		},
		{
			name: "top-level array format",
			payload: []byte(`[
				{
					"title": "Q1",
					"options": [
						{"content": "Opt1", "isCorrect": true},
						{"content": "Opt2", "isCorrect": false}
					]
				},
				{
					"title": "Q2",
					"options": [
						{"content": "OptA", "isCorrect": false},
						{"content": "OptB", "isCorrect": true}
					]
				}
			]`),
			createdBy:     2,
			expectedCount: 2,
			expectErr:     false,
		},
		{
			name:          "empty object payload",
			payload:       []byte(`{"questions":[]}`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     true,
			errContains:   "empty",
		},
		{
			name:          "empty array payload",
			payload:       []byte(`[]`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     true,
			errContains:   "empty",
		},
		{
			name:          "completely invalid JSON",
			payload:       []byte(`not json at all`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     true,
			errContains:   "invalid upload payload",
		},
		{
			name: "partially invalid questions - invalid ones skipped",
			payload: []byte(`{
				"questions": [
					{
						"title": "Valid Q",
						"options": [
							{"content": "A", "isCorrect": true},
							{"content": "B", "isCorrect": false}
						]
					},
					{
						"title": "Invalid - only one option",
						"options": [
							{"content": "Only", "isCorrect": true}
						]
					},
					{
						"title": "Invalid - blank option",
						"options": [
							{"content": "", "isCorrect": true},
							{"content": "B", "isCorrect": false}
						]
					},
					{
						"title": "Valid Q2",
						"options": [
							{"content": "X", "isCorrect": false},
							{"content": "Y", "isCorrect": true}
						]
					},
					{
						"title": "Invalid - zero correct",
						"options": [
							{"content": "A", "isCorrect": false},
							{"content": "B", "isCorrect": false}
						]
					}
				]
			}`),
			createdBy:     1,
			expectedCount: 2,
			expectErr:     false,
		},
		{
			name: "all questions invalid - returns zero no error",
			payload: []byte(`{
				"questions": [
					{
						"title": "Bad 1",
						"options": [{"content": "A", "isCorrect": false}]
					},
					{
						"title": "Bad 2",
						"options": [
							{"content": "A", "isCorrect": false},
							{"content": "B", "isCorrect": false}
						]
					}
				]
			}`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     false,
		},
		{
			name: "single valid question in object format",
			payload: []byte(`{
				"questions": [
					{
						"title": "Single",
						"description": "Just one",
						"options": [
							{"content": "Yes", "isCorrect": true},
							{"content": "No", "isCorrect": false}
						]
					}
				]
			}`),
			createdBy:     42,
			expectedCount: 1,
			expectErr:     false,
		},
		{
			name:          "null json",
			payload:       []byte(`null`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     true,
			errContains:   "empty",
		},
		{
			name:          "malformed JSON - missing bracket",
			payload:       []byte(`{"questions":[{"title":"Q","options":[{"content":"A","isCorrect":true}]`),
			createdBy:     1,
			expectedCount: 0,
			expectErr:     true,
			errContains:   "invalid upload payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			svc := NewQuestionService(db, testLogger())

			count, err := svc.UploadFromJSON(tt.payload, tt.createdBy)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("UploadFromJSON() expected error containing %q, got nil; count=%d", tt.errContains, count)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("UploadFromJSON() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("UploadFromJSON() unexpected error: %v", err)
			}
			if count != tt.expectedCount {
				t.Errorf("UploadFromJSON() count = %d, want %d", count, tt.expectedCount)
			}

			if count > 0 {
				var questionCount int64
				if err := db.Model(&models.Question{}).Count(&questionCount).Error; err != nil {
					t.Fatalf("failed to count questions: %v", err)
				}
				if int(questionCount) != tt.expectedCount {
					t.Errorf("database has %d questions, want %d", questionCount, tt.expectedCount)
				}
			}
		})
	}
}
