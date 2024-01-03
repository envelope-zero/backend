package models_test

import (
	"time"

	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func (suite *TestSuiteStandard) TestModelTimeUTC() {
	tz, _ := time.LoadLocation("Europe/Berlin")

	model := models.DefaultModel{
		Timestamps: models.Timestamps{
			CreatedAt: time.Date(2000, 1, 2, 3, 4, 5, 6, tz),
			UpdatedAt: time.Date(2001, 2, 3, 4, 5, 6, 7, tz),
			DeletedAt: &gorm.DeletedAt{Time: time.Now().In(tz)},
		},
	}

	err := model.AfterFind(models.DB)
	if err != nil {
		assert.Fail(suite.T(), "model.AfterFind failed")
	}

	assert.Equal(suite.T(), time.UTC, model.CreatedAt.Location(), "Timezone for model is not UTC")
	assert.Equal(suite.T(), time.UTC, model.UpdatedAt.Location(), "Timezone for model is not UTC")
	assert.Equal(suite.T(), time.UTC, model.DeletedAt.Time.Location(), "Timezone for model is not UTC")
}
