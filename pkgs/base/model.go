package base

import (
	"time"
)

type Model struct {
	ID        int   `gorm:"type:int(10) AUTO_INCREMENT;primarykey;autoIncrement;autoIncrementIncrement:1"`
	CreatedAt int64 `gorm:"type:int(10);autoCreateTime"`
	UpdatedAt int64 `gorm:"type:int(10);autoUpdateTime"`
	DeletedAt int64 `gorm:"type:int(10);index"`
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
