package model

import (
	"gorm.io/gorm"
)

type Execution struct {
	gorm.Model
	TaskName   string `gorm:"column:task_name"`
	TaskOwners string `gorm:"column:task_owners"`
	Status     string `gorm:"column:status"`
	RetryCount int    `gorm:"column:retry_count"`
	Payload    string `gorm:"column:payload"`
	Result     string `gorm:"column:result"`
}

func (t Execution) TableName() string {
	return "executions"
}
