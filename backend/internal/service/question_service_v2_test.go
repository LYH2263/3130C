package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type mockQuestionRepo struct {
	questions []models.Question
	createErr error
}

func (m *mockQuestionRepo) List() ([]models.Question, error) {
	return m.questions, nil
}

func (m *mockQuestionRepo) Create(q *models.Question) error {
	if m.createErr != nil {
		return m.createErr
	}
	q.ID = uint(len(m.questions) + 1)
	m.questions = append(m.questions, *q)
	return nil
}

func (m *mockQuestionRepo) GetByID(id uint) (*models.Question, error) {
	for _, q := range m.questions {
		if q.ID == id {
			return &q, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockQuestionRepo) Update(q *models.Question, opts []models.QuestionOption) error {
	return nil
}

func (m *mockQuestionRepo) Delete(id uint) error {
	return nil
}

func (m *mockQuestionRepo) GetQuiz(limit int) ([]models.Question, error) {
	if len(m.questions) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if limit > 0 && limit < len(m.questions) {
		return m.questions[:limit], nil
	}
	return m.questions, nil
}

func (m *mockQuestionRepo) Count() (int64, error) {
	return int64(len(m.questions)), nil
}

type mockLogger struct {
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.infoMsgs = append(m.infoMsgs, msg)
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.warnMsgs = append(m.warnMsgs, msg)
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.errorMsgs = append(m.errorMsgs, msg)
}

func TestQuestionServiceV2_CreateQuestion_WithMock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     dto.QuestionInput
		createdBy uint
		wantErr   bool
		errType   error
	}{
		{
			name: "valid question",
			input: dto.QuestionInput{
				Title: "Test Q",
				Options: []dto.QuestionOptionInput{
					{Content: "A", IsCorrect: true},
					{Content: "B", IsCorrect: false},
				},
			},
			createdBy: 1,
			wantErr:   false,
		},
		{
			name: "invalid question - only one option",
			input: dto.QuestionInput{
				Title: "Bad Q",
				Options: []dto.QuestionOptionInput{
					{Content: "A", IsCorrect: true},
				},
			},
			createdBy: 1,
			wantErr:   true,
			errType:   ErrInvalidQuestion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := &mockQuestionRepo{}
			mockLog := &mockLogger{}
			svc := NewQuestionServiceV2(mockRepo, mockLog)

			result, err := svc.CreateQuestion(tt.input, tt.createdBy)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Fatalf("error = %v, want %v", err, tt.errType)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID == 0 {
				t.Error("expected non-zero ID")
			}
			if len(mockLog.infoMsgs) != 1 {
				t.Errorf("expected 1 info log, got %d", len(mockLog.infoMsgs))
			}
		})
	}
}

func TestQuestionServiceV2_CreateQuestion_DBError(t *testing.T) {
	t.Parallel()

	mockRepo := &mockQuestionRepo{
		createErr: errors.New("db connection failed"),
	}
	mockLog := &mockLogger{}
	svc := NewQuestionServiceV2(mockRepo, mockLog)

	input := dto.QuestionInput{
		Title: "Test Q",
		Options: []dto.QuestionOptionInput{
			{Content: "A", IsCorrect: true},
			{Content: "B", IsCorrect: false},
		},
	}

	_, err := svc.CreateQuestion(input, 1)

	if err == nil {
		t.Fatal("expected error from DB, got nil")
	}
	if err.Error() != "db connection failed" {
		t.Errorf("error message mismatch: got %q", err.Error())
	}
}
