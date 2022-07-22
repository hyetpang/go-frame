package base

type Model struct {
	ID        int   `gorm:"type:int(10) AUTO_INCREMENT;primaryKey;autoIncrement;autoIncrementIncrement:1" json:"id"`
	CreatedAt int64 `gorm:"type:int(10);autoCreateTime" json:"created_at"`
	UpdatedAt int64 `gorm:"type:int(10);autoUpdateTime" json:"updated_at"`
	DeletedAt int64 `gorm:"type:int(10);index" json:"deleted_at"`
}
