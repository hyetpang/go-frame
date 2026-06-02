package pagination

type PaginationI interface {
	GetOffset() int
	GetPageSize() int
}

type Pagination struct {
	Page     int `json:"page" form:"page" binding:"min=1"`                   // 页码,最小值1
	PageSize int `json:"page_size" form:"page_size" binding:"min=0,max=200"` // 可不传,默认20,上限200
}

const (
	defaultPageSize = 20
	// maxPageSize 限制单页最大条数,防止客户端传入超大值直接变成 gorm 的 LIMIT,
	// 造成内存/DB 资源耗尽的 DoS。binding 已有 max=200 拦截,这里在 getter 内再 clamp 一次做双保险,
	// 覆盖未走 binding 校验(如手动构造 Pagination)的调用路径。
	maxPageSize = 200
)

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
	// clamp 到上限,避免超大 PageSize 直接传给 gorm LIMIT 导致资源耗尽
	if op.PageSize > maxPageSize {
		return maxPageSize
	}
	return op.PageSize
}
