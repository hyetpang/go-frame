package base

import (
	"time"
)

type Model struct {
	ID        int   `gorm:"type:int(10) AUTO_INCREMENT;primarykey;autoIncrement;autoIncrementIncrement:1"`
	CreatedAt int64 `gorm:"type:int(10);default:0"`
	UpdatedAt int64 `gorm:"type:int(10);default:0"`
	DeletedAt int64 `gorm:"type:int(10);default:0;index"`
}

func NewModel() Model {
	now := time.Now().Unix()
	return Model{
		ID:        0,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: 0,
	}
}
