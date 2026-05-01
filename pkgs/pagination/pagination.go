package pagination

type PaginationI interface {
	GetOffset() int
	GetPageSize() int
}

type Pagination struct {
	Page     int `json:"page" form:"page" binding:"min=1"`           // 页码,最小值1
	PageSize int `json:"page_size" form:"page_size" binding:"min=0"` // 可不传,默认20
}

const defaultPageSize = 20

func (op *Pagination) GetOffset() int {
	// Page < 1 时统一返回 0，避免向 gorm 传入负数 offset
	if op.Page < 1 {
		return 0
	}
	return (op.Page - 1) * op.GetPageSize()
}

func (op *Pagination) GetPageSize() int {
	if op.PageSize <= 0 {
		return defaultPageSize
	}
	return op.PageSize
}
