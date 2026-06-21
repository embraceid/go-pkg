package pagination

import "testing"

func TestNewPaginationAppliesDefaults(t *testing.T) {
	p := NewPagination(0, 0)

	if p.Page != DefaultPage {
		t.Fatalf("expected default page %d, got %d", DefaultPage, p.Page)
	}
	if p.Limit != DefaultLimit {
		t.Fatalf("expected default limit %d, got %d", DefaultLimit, p.Limit)
	}
}

func TestNewPaginationWithMaxUsesMaximumLimit(t *testing.T) {
	p := NewPaginationWithMax(2)

	if p.Page != 2 {
		t.Fatalf("expected page 2, got %d", p.Page)
	}
	if p.Limit != MaximumLimit {
		t.Fatalf("expected maximum limit %d, got %d", MaximumLimit, p.Limit)
	}
}

func TestValidateResetsOutOfRangeLimit(t *testing.T) {
	p := (&Pagination{Page: 1, Limit: MaximumLimit + 1}).Validate()

	if p.Limit != DefaultLimit {
		t.Fatalf("expected default limit %d, got %d", DefaultLimit, p.Limit)
	}
}

func TestSetPaginationSetsTotalAndTotalPage(t *testing.T) {
	p := NewPagination(2, 10)

	p.SetPagination(25)

	if p.Total != 25 {
		t.Fatalf("expected total 25, got %d", p.Total)
	}
	if p.TotalPage != 3 {
		t.Fatalf("expected total page 3, got %d", p.TotalPage)
	}
}

func TestSetPaginationSkipsTotalPageWhenTotalZero(t *testing.T) {
	p := NewPagination(1, 10)

	p.SetPagination()

	if p.TotalPage != 0 {
		t.Fatalf("expected total page 0, got %d", p.TotalPage)
	}
}

func TestGetOffsetUsesValidatedPageAndLimit(t *testing.T) {
	p := NewPagination(3, 10)

	if got := p.GetOffset(); got != 20 {
		t.Fatalf("expected offset 20, got %d", got)
	}
}

func TestHasNextPageReportsMorePages(t *testing.T) {
	p := NewPagination(2, 10)
	p.SetPagination(25)

	if !p.HasNextPage() {
		t.Fatal("expected next page to exist")
	}
}

func TestIncrAdvancesPage(t *testing.T) {
	p := NewPagination(1, 10)

	p.Incr()

	if p.Page != 2 {
		t.Fatalf("expected page 2, got %d", p.Page)
	}
}
