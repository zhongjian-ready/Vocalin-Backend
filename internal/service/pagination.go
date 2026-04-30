package service

type Pagination struct {
	Page     int
	PageSize int
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type PaginatedResult[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

func NewPagination(page, pageSize int) Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Pagination{Page: page, PageSize: pageSize}
}

func NewPaginatedResult[T any](items []T, pagination Pagination, total int) PaginatedResult[T] {
	totalPages := 0
	if pagination.PageSize > 0 {
		totalPages = (total + pagination.PageSize - 1) / pagination.PageSize
	}
	return PaginatedResult[T]{
		Items:      items,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
