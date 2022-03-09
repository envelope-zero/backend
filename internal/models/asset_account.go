package models

// AssetAccount represents an asset account, e.g. a bank account
type AssetAccount struct {
	Model
	Name     string `json:"name"`
	BudgetID int
	Budget   Budget
}

// CreateAssetAccount defines all values required to create an new account
type CreateAssetAccount struct {
	Name string `json:"name" binding:"required"`
}
