package pagination

import "math"

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaximumLimit = 100
)

type Pagination struct {
	Page      int
	Limit     int
	Total     int
	TotalPage int
}

func NewPagination(page int, limit int) *Pagination {
	p := &Pagination{Page: page, Limit: limit}
	return p.Validate()
}

func NewPaginationWithMax(page int) *Pagination {
	p := &Pagination{Page: page, Limit: MaximumLimit}
	return p.Validate()
}

func (p *Pagination) Validate() *Pagination {
	if p.Page <= 0 {
		p.Page = DefaultPage
	}
	if p.Limit <= 0 || p.Limit > MaximumLimit {
		p.Limit = DefaultLimit
	}
	return p
}

func (p *Pagination) SetPagination(total ...int) {
	if len(total) != 0 {
		p.Total = total[0]
	}
	if p.Total == 0 {
		return
	}
	p.TotalPage = int(math.Ceil(float64(p.Total) / float64(p.Limit)))
}

func (p *Pagination) GetOffset() int {
	var offset int
	if p.Page > 0 {
		offset = p.Limit * (p.Page - 1)
	}
	return offset
}

func (p *Pagination) HasNextPage() bool {
	return p.Page < p.TotalPage
}

func (p *Pagination) Incr() {
	p.Page++
}
