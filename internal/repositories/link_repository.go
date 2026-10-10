package repositories

import (
	"gorm.io/gorm"

	"kliktaut-api/internal/models"
)

type LinkRepository struct {
	DB *gorm.DB
}

func NewLinkRepository(db *gorm.DB) *LinkRepository {
	return &LinkRepository{DB: db}
}

func (r *LinkRepository) Create(link *models.Link) error {
	return r.DB.Create(link).Error
}

func (r *LinkRepository) GetByShortCode(code string) (*models.Link, error) {
	var link models.Link
	err := r.DB.Where("short_code = ?", code).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *LinkRepository) GetByID(id string) (*models.Link, error) {
	var link models.Link
	err := r.DB.Where("id = ?", id).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// List mengembalikan links + exact total untuk meta pagination.
// search: ILIKE substring pada title OR short_code (keputusan OQ-005).
func (r *LinkRepository) List(page, perPage int, search string) ([]models.Link, int64, error) {
	query := r.DB.Model(&models.Link{})
	if search != "" {
		term := "%" + search + "%"
		query = query.Where("title ILIKE ? OR short_code ILIKE ?", term, term)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var links []models.Link
	err := query.
		Order("created_at DESC").
		Limit(perPage).
		Offset((page - 1) * perPage).
		Find(&links).Error
	if err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

// UpdateFields hanya menyentuh kolom yang dikirim (patch sebagian).
func (r *LinkRepository) UpdateFields(id string, fields map[string]interface{}) error {
	return r.DB.Model(&models.Link{}).Where("id = ?", id).Updates(fields).Error
}

func (r *LinkRepository) ExistsShortCode(code string) (bool, error) {
	var count int64
	err := r.DB.Model(&models.Link{}).Where("short_code = ?", code).Count(&count).Error
	return count > 0, err
}
