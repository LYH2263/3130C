package service

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type AttemptService struct {
	db  *gorm.DB
	log *slog.Logger
}

type SubmitResult struct {
	AttemptID uint   `json:"attemptId"`
	Score     int    `json:"score"`
	Total     int    `json:"total"`
	Rate      string `json:"rate"`
}

type StudentMistake struct {
	QuestionID    uint   `json:"questionId"`
	Title         string `json:"title"`
	WrongCount    int64  `json:"wrongCount"`
	CorrectOption string `json:"correctOption"`
}

type ClassWrongStat struct {
	ClassID    uint   `json:"classId"`
	ClassName  string `json:"className"`
	QuestionID uint   `json:"questionId"`
	Question   string `json:"question"`
	WrongCount int64  `json:"wrongCount"`
}

type RecentAttempt struct {
	ID        uint   `json:"id"`
	Student   string `json:"student"`
	ClassName string `json:"className"`
	Score     int    `json:"score"`
	Total     int    `json:"total"`
	CreatedAt string `json:"createdAt"`
}

type Overview struct {
	StudentCount  int64 `json:"studentCount"`
	ClassCount    int64 `json:"classCount"`
	QuestionCount int64 `json:"questionCount"`
	AttemptCount  int64 `json:"attemptCount"`
}

func NewAttemptService(db *gorm.DB, log *slog.Logger) *AttemptService {
	return &AttemptService{db: db, log: log}
}

func (s *AttemptService) Submit(userID uint, classID uint, req dto.SubmitRequest) (*SubmitResult, error) {
	if len(req.Answers) == 0 {
		return nil, ErrInvalidSubmission
	}

	questionIDs := uniqueQuestionIDs(req.Answers)
	var questions []models.Question
	if err := s.db.Preload("Options").Where("id IN ?", questionIDs).Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}
	if len(questions) == 0 {
		return nil, ErrNoQuestions
	}

	questionMap := make(map[uint]models.Question, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	answersModel := make([]models.AttemptAnswer, 0, len(req.Answers))
	score := 0

	for _, answer := range req.Answers {
		question, ok := questionMap[answer.QuestionID]
		if !ok {
			return nil, ErrInvalidSubmission
		}

		selectedValid := false
		correct := false
		for _, opt := range question.Options {
			if opt.ID == answer.OptionID {
				selectedValid = true
				if opt.IsCorrect {
					correct = true
				}
				break
			}
		}
		if !selectedValid {
			return nil, ErrInvalidSubmission
		}
		if correct {
			score++
		}

		answersModel = append(answersModel, models.AttemptAnswer{
			QuestionID:       answer.QuestionID,
			SelectedOptionID: answer.OptionID,
			IsCorrect:        correct,
		})
	}

	total := len(req.Answers)
	attempt := models.Attempt{
		UserID:  userID,
		ClassID: classID,
		Score:   score,
		Total:   total,
		Answers: answersModel,
	}
	if err := s.db.Create(&attempt).Error; err != nil {
		return nil, fmt.Errorf("save attempt: %w", err)
	}

	rate := fmt.Sprintf("%.0f%%", (float64(score)/float64(total))*100)
	s.log.Info("attempt submitted", "attemptID", attempt.ID, "userID", userID, "score", score, "total", total)

	return &SubmitResult{AttemptID: attempt.ID, Score: score, Total: total, Rate: rate}, nil
}

func (s *AttemptService) StudentAttempts(userID uint) ([]models.Attempt, error) {
	var attempts []models.Attempt
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("load student attempts: %w", err)
	}
	return attempts, nil
}

func (s *AttemptService) StudentMistakes(userID uint) ([]StudentMistake, error) {
	var attempts []models.Attempt
	if err := s.db.Where("user_id = ?", userID).Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("load attempts: %w", err)
	}
	if len(attempts) == 0 {
		return []StudentMistake{}, nil
	}

	attemptIDs := make([]uint, 0, len(attempts))
	for _, a := range attempts {
		attemptIDs = append(attemptIDs, a.ID)
	}

	var wrongAnswers []models.AttemptAnswer
	if err := s.db.Where("attempt_id IN ? AND is_correct = ?", attemptIDs, false).Find(&wrongAnswers).Error; err != nil {
		return nil, fmt.Errorf("load wrong answers: %w", err)
	}
	if len(wrongAnswers) == 0 {
		return []StudentMistake{}, nil
	}

	wrongCountMap := map[uint]int64{}
	for _, item := range wrongAnswers {
		wrongCountMap[item.QuestionID]++
	}

	questionIDs := make([]uint, 0, len(wrongCountMap))
	for id := range wrongCountMap {
		questionIDs = append(questionIDs, id)
	}
	var questions []models.Question
	if err := s.db.Preload("Options").Where("id IN ?", questionIDs).Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load mistake questions: %w", err)
	}

	result := make([]StudentMistake, 0, len(questions))
	for _, q := range questions {
		correct := ""
		for _, opt := range q.Options {
			if opt.IsCorrect {
				correct = opt.Content
				break
			}
		}
		result = append(result, StudentMistake{
			QuestionID:    q.ID,
			Title:         q.Title,
			WrongCount:    wrongCountMap[q.ID],
			CorrectOption: correct,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].WrongCount > result[j].WrongCount
	})
	return result, nil
}

type classWrongStatRow struct {
	ClassID    uint   `gorm:"column:class_id"`
	ClassName  string `gorm:"column:class_name"`
	QuestionID uint   `gorm:"column:question_id"`
	Question   string `gorm:"column:question_title"`
	WrongCount int64  `gorm:"column:wrong_count"`
}

func (s *AttemptService) ClassWrongStats(limit int) ([]ClassWrongStat, error) {
	rows, err := s.queryClassWrongStats(limit)
	if err != nil {
		return nil, fmt.Errorf("query class wrong stats: %w", err)
	}
	if len(rows) == 0 {
		return []ClassWrongStat{}, nil
	}
	return buildClassWrongStats(rows), nil
}

// queryClassWrongStats 执行班级错题热区的数据库聚合查询。
//
// 推荐索引（提升查询性能）：
//   - attempt_answers(is_correct, question_id, attempt_id)  -- 覆盖过滤与分组
//   - attempts(id, class_id)                                -- 覆盖 JOIN 键
//   - class_rooms(id, name)                                 -- 覆盖班级名查询
//   - questions(id, title)                                  -- 覆盖题目标题查询
func (s *AttemptService) queryClassWrongStats(limit int) ([]classWrongStatRow, error) {
	db := s.db.
		Table("attempt_answers aa").
		Select(
			"a.class_id AS class_id, " +
				"COALESCE(cr.name, '') AS class_name, " +
				"aa.question_id AS question_id, " +
				"COALESCE(q.title, '') AS question_title, " +
				"COUNT(*) AS wrong_count",
		).
		Joins("JOIN attempts a ON a.id = aa.attempt_id").
		Joins("LEFT JOIN class_rooms cr ON cr.id = a.class_id").
		Joins("LEFT JOIN questions q ON q.id = aa.question_id").
		Where("aa.is_correct = ?", false).
		Group("a.class_id, aa.question_id").
		Order("wrong_count DESC, a.class_id ASC, aa.question_id ASC")

	if limit > 0 {
		db = db.Limit(limit)
	}

	var rows []classWrongStatRow
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func buildClassWrongStats(rows []classWrongStatRow) []ClassWrongStat {
	result := make([]ClassWrongStat, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClassWrongStat{
			ClassID:    row.ClassID,
			ClassName:  row.ClassName,
			QuestionID: row.QuestionID,
			Question:   row.Question,
			WrongCount: row.WrongCount,
		})
	}
	return result
}

func (s *AttemptService) TeacherRecentAttempts(limit int) ([]RecentAttempt, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var attempts []models.Attempt
	if err := s.db.Preload("User").Preload("ClassRoom").Order("created_at desc").Limit(limit).Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("load recent attempts: %w", err)
	}

	result := make([]RecentAttempt, 0, len(attempts))
	for _, item := range attempts {
		result = append(result, RecentAttempt{
			ID:        item.ID,
			Student:   item.User.Username,
			ClassName: item.ClassRoom.Name,
			Score:     item.Score,
			Total:     item.Total,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func (s *AttemptService) Overview() (*Overview, error) {
	result := &Overview{}
	if err := s.db.Model(&models.User{}).Where("role = ?", models.RoleStudent).Count(&result.StudentCount).Error; err != nil {
		return nil, fmt.Errorf("count students: %w", err)
	}
	if err := s.db.Model(&models.ClassRoom{}).Count(&result.ClassCount).Error; err != nil {
		return nil, fmt.Errorf("count classes: %w", err)
	}
	if err := s.db.Model(&models.Question{}).Count(&result.QuestionCount).Error; err != nil {
		return nil, fmt.Errorf("count questions: %w", err)
	}
	if err := s.db.Model(&models.Attempt{}).Count(&result.AttemptCount).Error; err != nil {
		return nil, fmt.Errorf("count attempts: %w", err)
	}
	return result, nil
}

func uniqueQuestionIDs(items []dto.SubmitAnswerItem) []uint {
	set := map[uint]struct{}{}
	for _, item := range items {
		set[item.QuestionID] = struct{}{}
	}
	ids := make([]uint, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
