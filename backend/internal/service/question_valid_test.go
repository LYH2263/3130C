package service

import (
	"testing"

	"label3130/backend/internal/dto"
)

func TestIsQuestionValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		options  []dto.QuestionOptionInput
		expected bool
	}{
		{
			name:     "two options exactly one correct",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: true}, {Content: "B", IsCorrect: false}},
			expected: true,
		},
		{
			name:     "three options exactly one correct",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: false}, {Content: "B", IsCorrect: true}, {Content: "C", IsCorrect: false}},
			expected: true,
		},
		{
			name:     "options count less than 2",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: true}},
			expected: false,
		},
		{
			name:     "empty options",
			options:  []dto.QuestionOptionInput{},
			expected: false,
		},
		{
			name:     "nil options",
			options:  nil,
			expected: false,
		},
		{
			name:     "blank option content",
			options:  []dto.QuestionOptionInput{{Content: "", IsCorrect: true}, {Content: "B", IsCorrect: false}},
			expected: false,
		},
		{
			name:     "whitespace only option content",
			options:  []dto.QuestionOptionInput{{Content: "   ", IsCorrect: true}, {Content: "B", IsCorrect: false}},
			expected: false,
		},
		{
			name:     "blank option in middle",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: false}, {Content: "  \t\n  ", IsCorrect: true}, {Content: "C", IsCorrect: false}},
			expected: false,
		},
		{
			name:     "zero correct options",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: false}, {Content: "B", IsCorrect: false}},
			expected: false,
		},
		{
			name:     "multiple correct options",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: true}, {Content: "B", IsCorrect: true}},
			expected: false,
		},
		{
			name:     "three options two correct",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: true}, {Content: "B", IsCorrect: false}, {Content: "C", IsCorrect: true}},
			expected: false,
		},
		{
			name:     "all correct options",
			options:  []dto.QuestionOptionInput{{Content: "A", IsCorrect: true}, {Content: "B", IsCorrect: true}, {Content: "C", IsCorrect: true}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isQuestionValid(tt.options)

			if got != tt.expected {
				t.Errorf("isQuestionValid() = %v, want %v; options=%+v", got, tt.expected, tt.options)
			}
		})
	}
}
