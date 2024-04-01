package models

import (
	"encoding/json"

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

// Returns all match rules on this instance for export
func (MatchRule) Export() (json.RawMessage, error) {
	var matchRules []MatchRule
	err := DB.Unscoped().Where(&MatchRule{}).Find(&matchRules).Error
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(&matchRules)
	if err != nil {
		return json.RawMessage{}, err
	}
	return json.RawMessage(j), nil
}
