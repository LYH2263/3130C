package service

import (
	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

// QuestionRepository 定义题目数据访问接口，用于解耦 service 与具体 DB 实现。
// 通过接口抽象，测试时可注入 mock 实现，无需真实数据库。
type QuestionRepository interface {
	List() ([]models.Question, error)
	Create(question *models.Question) error
	GetByID(id uint) (*models.Question, error)
	Update(question *models.Question, options []models.QuestionOption) error
	Delete(id uint) error
	GetQuiz(limit int) ([]models.Question, error)
	Count() (int64, error)
}

// AttemptRepository 定义答题记录数据访问接口。
type AttemptRepository interface {
	Create(attempt *models.Attempt) error
	GetByUser(userID uint) ([]models.Attempt, error)
	GetQuestionsByIDs(ids []uint) ([]models.Question, error)
}

// GormQuestionRepository 是基于 GORM 的 QuestionRepository 实现。
type GormQuestionRepository struct {
	db *gorm.DB
}

// NewGormQuestionRepository 创建基于 GORM 的题目仓储。
func NewGormQuestionRepository(db *gorm.DB) *GormQuestionRepository {
	return &GormQuestionRepository{db: db}
}

func (r *GormQuestionRepository) List() ([]models.Question, error) {
	var questions []models.Question
	if err := r.db.Preload("Options").Order("id desc").Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *GormQuestionRepository) Create(question *models.Question) error {
	return r.db.Create(question).Error
}

func (r *GormQuestionRepository) GetByID(id uint) (*models.Question, error) {
	var question models.Question
	if err := r.db.Preload("Options").First(&question, id).Error; err != nil {
		return nil, err
	}
	return &question, nil
}

func (r *GormQuestionRepository) Update(question *models.Question, options []models.QuestionOption) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(question).Updates(map[string]any{
			"title":       question.Title,
			"description": question.Description,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("question_id = ?", question.ID).Delete(&models.QuestionOption{}).Error; err != nil {
			return err
		}
		for i := range options {
			options[i].QuestionID = question.ID
		}
		if err := tx.Create(&options).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *GormQuestionRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionOption{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&models.Question{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *GormQuestionRepository) GetQuiz(limit int) ([]models.Question, error) {
	var questions []models.Question
	query := r.db.Preload("Options").Order("id asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *GormQuestionRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Question{}).Count(&count).Error
	return count, err
}

// GormAttemptRepository 是基于 GORM 的 AttemptRepository 实现。
type GormAttemptRepository struct {
	db *gorm.DB
}

// NewGormAttemptRepository 创建基于 GORM 的答题记录仓储。
func NewGormAttemptRepository(db *gorm.DB) *GormAttemptRepository {
	return &GormAttemptRepository{db: db}
}

func (r *GormAttemptRepository) Create(attempt *models.Attempt) error {
	return r.db.Create(attempt).Error
}

func (r *GormAttemptRepository) GetByUser(userID uint) ([]models.Attempt, error) {
	var attempts []models.Attempt
	if err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *GormAttemptRepository) GetQuestionsByIDs(ids []uint) ([]models.Question, error) {
	var questions []models.Question
	if err := r.db.Preload("Options").Where("id IN ?", ids).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

// QuestionServiceV2 使用接口依赖注入的版本，便于单元测试时 mock DB。
// 构造时传入 repository 接口而非具体的 *gorm.DB。
type QuestionServiceV2 struct {
	repo QuestionRepository
	log  LoggerInterface
}

// LoggerInterface 日志接口，进一步解耦具体日志实现。
type LoggerInterface interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NewQuestionServiceV2 使用依赖注入构造 QuestionService。
// 好处：测试时可传入 mock QuestionRepository 与 mock Logger，
// 无需启动真实数据库，测试更快且更稳定。
func NewQuestionServiceV2(repo QuestionRepository, log LoggerInterface) *QuestionServiceV2 {
	return &QuestionServiceV2{repo: repo, log: log}
}

// CreateQuestion 是使用接口版本的创建逻辑，纯业务逻辑与数据访问解耦。
func (s *QuestionServiceV2) CreateQuestion(input dto.QuestionInput, createdBy uint) (*models.Question, error) {
	if !isQuestionValid(input.Options) {
		return nil, ErrInvalidQuestion
	}

	question := models.Question{
		Title:       input.Title,
		Description: input.Description,
		CreatedBy:   createdBy,
		Options:     toOptionModels(input.Options),
	}

	if err := s.repo.Create(&question); err != nil {
		return nil, err
	}

	s.log.Info("question created", "questionID", question.ID, "createdBy", createdBy)
	return &question, nil
}
