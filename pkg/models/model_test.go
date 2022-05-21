package models_test

import (
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestModelTimeUTC(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Berlin")

	model := models.Model{
		CreatedAt: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
		UpdatedAt: time.Date(2001, 2, 3, 4, 5, 6, 7, tz),
		DeletedAt: &gorm.DeletedAt{Time: time.Now().In(tz)},
	}

	err := model.AfterFind(models.DB)
	if err != nil {
		assert.Fail(t, "model.AfterFind failed")
	}

	assert.Equal(t, time.UTC, model.CreatedAt.Location(), "Timezone for model is not UTC")
	assert.Equal(t, time.UTC, model.UpdatedAt.Location(), "Timezone for model is not UTC")
	assert.Equal(t, time.UTC, model.DeletedAt.Time.Location(), "Timezone for model is not UTC")
}
