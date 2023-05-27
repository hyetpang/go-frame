package pagination

type PaginationI interface {
	GetPage() int
	GetPageSize() int
}

type Pagination struct {
	Page     int `json:"page" form:"page" binding:"min=1"` // 页码,最小值1
	PageSize int `json:"page_size" form:"page_size"`       // 可不传,默认20
}

func (op *Pagination) GetPage() int {
	return (op.Page - 1) * op.GetPageSize()
}
const defaultMaxPageSize = 20

func (op *Pagination) GetPageSize() int {
	if op.PageSize == 0 {
		op.PageSize = defaultMaxPageSize
	}
	return op.PageSize
}
