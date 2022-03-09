package models

// AssetAccount represents an asset account, e.g. a bank account
type AssetAccount struct {
	Model
	Name     string `json:"name"`
	BudgetID int    `json:"budgetId"`
}

// CreateAssetAccount defines all values required to create a new asset account
type CreateAssetAccount struct {
	Name string `json:"name" binding:"required"`
}
