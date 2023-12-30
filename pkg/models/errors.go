package models

import "errors"

var (
	ErrAllocationZero        = errors.New("allocation amounts must be non-zero. Instead of setting to zero, delete the Allocation")
	ErrGoalAmountNotPositive = errors.New("goal amounts must be larger than zero")
)
