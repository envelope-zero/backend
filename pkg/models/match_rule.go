package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchRule struct {
	DefaultModel
	AccountID uuid.UUID
	Priority  uint
	Match     string
}

func (m *MatchRule) BeforeCreate(tx *gorm.DB) error {
	_ = m.DefaultModel.BeforeCreate(tx)

	toSave := tx.Statement.Dest.(*MatchRule)
	return m.checkIntegrity(tx, *toSave)
}

func (m *MatchRule) BeforeUpdate(tx *gorm.DB) (err error) {
	if tx.Statement.Changed("AccountID") {
		toSave := tx.Statement.Dest.(MatchRule)
		err := m.checkIntegrity(tx, toSave)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkIntegrity verifies references to other resources
func (m *MatchRule) checkIntegrity(tx *gorm.DB, toSave MatchRule) error {
	return tx.First(&Account{}, toSave.AccountID).Error
}
