package base

type Model struct {
	ID        int   `gorm:"type:bigint;primaryKey;autoIncrement" json:"id"`
	CreatedAt int64 `gorm:"type:bigint;autoCreateTime" json:"created_at"`
	UpdatedAt int64 `gorm:"type:bigint;autoUpdateTime" json:"updated_at"`
	DeletedAt int64 `gorm:"type:bigint;index" json:"deleted_at"`
}
