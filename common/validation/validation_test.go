package validation

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBagCollectsFailuresInOrder(t *testing.T) {
	t.Parallel()

	bag := NewBag()
	bag.RequiredString("code", "", "code is required")
	bag.MaxStringLength("name", strings.Repeat("A", 51), 50, "name must be at most 50 characters")

	require.True(t, bag.HasAny())
	require.Equal(t, 2, bag.Len())

	failures := bag.Failures()
	require.Equal(t, RequiredCode, failures[0].Code)
	require.Equal(t, "code", failures[0].Field)
	require.Equal(t, "code is required", failures[0].Message)

	require.Equal(t, MaxLengthCode, failures[1].Code)
	require.Equal(t, "name", failures[1].Field)
	require.Equal(t, "name must be at most 50 characters", failures[1].Message)
	require.Equal(t, []Param{{Key: "max", Value: "50"}}, failures[1].Params)
}

func TestBagReturnsCopiedFailures(t *testing.T) {
	t.Parallel()

	bag := NewBag()
	bag.RequiredString("code", "", "code is required")

	failures := bag.Failures()
	failures[0].Field = "mutated"

	require.Equal(t, "code", bag.Failures()[0].Field)
}

func TestBagCopiesParamsOnAdd(t *testing.T) {
	t.Parallel()

	bag := NewBag()
	params := []Param{{Key: "max", Value: "50"}}
	bag.Add(MaxLengthCode, "name", "name must be at most 50 characters", params...)
	params[0] = Param{Key: "mutated", Value: "999"}

	require.Equal(t, []Param{{Key: "max", Value: "50"}}, bag.Failures()[0].Params)
}

func TestBagReturnsCopiedFailureParams(t *testing.T) {
	t.Parallel()

	bag := NewBag()
	bag.Add(MaxLengthCode, "name", "name must be at most 50 characters", Param{Key: "max", Value: "50"})

	failures := bag.Failures()
	failures[0].Params[0] = Param{Key: "mutated", Value: "999"}

	require.Equal(t, []Param{{Key: "max", Value: "50"}}, bag.Failures()[0].Params)
}

func TestPrimitiveRules(t *testing.T) {
	t.Parallel()

	rolePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	tests := []struct {
		name      string
		validate  func(*Bag) bool
		wantCode  Code
		wantField string
		wantParam []Param
	}{
		{
			name:      "required string rejects empty string",
			validate:  func(bag *Bag) bool { return bag.RequiredString("code", "", "code is required") },
			wantCode:  RequiredCode,
			wantField: "code",
		},
		{
			name:      "required string rejects blank string",
			validate:  func(bag *Bag) bool { return bag.RequiredString("code", "   ", "code is required") },
			wantCode:  RequiredCode,
			wantField: "code",
		},
		{
			name:      "minimum string length rejects short values",
			validate:  func(bag *Bag) bool { return bag.MinStringLength("code", "A", 2, "code must be at least 2 characters") },
			wantCode:  MinLengthCode,
			wantField: "code",
			wantParam: []Param{{Key: "min", Value: "2"}},
		},
		{
			name: "minimum string length counts unicode characters",
			validate: func(bag *Bag) bool {
				return bag.MinStringLength("name", "東京", 3, "name must be at least 3 characters")
			},
			wantCode:  MinLengthCode,
			wantField: "name",
			wantParam: []Param{{Key: "min", Value: "3"}},
		},
		{
			name:      "maximum string length rejects long values",
			validate:  func(bag *Bag) bool { return bag.MaxStringLength("code", "ABC", 2, "code must be at most 2 characters") },
			wantCode:  MaxLengthCode,
			wantField: "code",
			wantParam: []Param{{Key: "max", Value: "2"}},
		},
		{
			name: "maximum string length counts unicode characters",
			validate: func(bag *Bag) bool {
				return bag.MaxStringLength("emoji", "🍜🍣🍱", 2, "emoji must be at most 2 characters")
			},
			wantCode:  MaxLengthCode,
			wantField: "emoji",
			wantParam: []Param{{Key: "max", Value: "2"}},
		},
		{
			name:      "minimum int rejects small values",
			validate:  func(bag *Bag) bool { return bag.MinInt("page", 0, 1, "page must be at least 1") },
			wantCode:  MinValueCode,
			wantField: "page",
			wantParam: []Param{{Key: "min", Value: "1"}},
		},
		{
			name:      "maximum int rejects large values",
			validate:  func(bag *Bag) bool { return bag.MaxInt("pageSize", 101, 100, "pageSize must be at most 100") },
			wantCode:  MaxValueCode,
			wantField: "pageSize",
			wantParam: []Param{{Key: "max", Value: "100"}},
		},
		{
			name:      "pattern rejects non-matching values",
			validate:  func(bag *Bag) bool { return bag.Pattern("code", "ADMIN ROLE", rolePattern, "code format is invalid") },
			wantCode:  PatternCode,
			wantField: "code",
		},
		{
			name: "one of rejects values outside allowed set",
			validate: func(bag *Bag) bool {
				return bag.OneOfString("status", "archived", "status is invalid", []string{"active", "inactive"})
			},
			wantCode:  OneOfCode,
			wantField: "status",
		},
		{
			name:      "ulid rejects invalid values",
			validate:  func(bag *Bag) bool { return bag.ULID("id", "bad-id", "id must be a valid ULID") },
			wantCode:  ULIDCode,
			wantField: "id",
		},
		{
			name:      "ulid rejects values starting with eight",
			validate:  func(bag *Bag) bool { return bag.ULID("id", "81HZ0000000000000000000001", "id must be a valid ULID") },
			wantCode:  ULIDCode,
			wantField: "id",
		},
		{
			name:      "ulid rejects values starting with z",
			validate:  func(bag *Bag) bool { return bag.ULID("id", "Z1HZ0000000000000000000001", "id must be a valid ULID") },
			wantCode:  ULIDCode,
			wantField: "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bag := NewBag()
			valid := tt.validate(bag)

			require.False(t, valid)
			require.Equal(t, 1, bag.Len())

			failure := bag.Failures()[0]
			require.Equal(t, tt.wantCode, failure.Code)
			require.Equal(t, tt.wantField, failure.Field)
			require.Equal(t, tt.wantParam, failure.Params)
		})
	}
}

func TestPrimitiveRulesAcceptValidValues(t *testing.T) {
	t.Parallel()

	bag := NewBag()
	rolePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	require.True(t, bag.RequiredString("code", "ADMIN", "code is required"))
	require.True(t, bag.MinStringLength("code", "ADMIN", 2, "code must be at least 2 characters"))
	require.True(t, bag.MinStringLength("name", "東京", 2, "name must be at least 2 characters"))
	require.True(t, bag.MaxStringLength("code", "ADMIN", 50, "code must be at most 50 characters"))
	require.True(t, bag.MaxStringLength("emoji", "🍜🍣", 2, "emoji must be at most 2 characters"))
	require.True(t, bag.MinInt("page", 1, 1, "page must be at least 1"))
	require.True(t, bag.MaxInt("pageSize", 100, 100, "pageSize must be at most 100"))
	require.True(t, bag.Pattern("code", "ADMIN_ROLE", rolePattern, "code format is invalid"))
	require.True(t, bag.OneOfString("status", "active", "status is invalid", []string{"active", "inactive"}))
	require.True(t, bag.ULID("id", "01HZ0000000000000000000001", "id must be a valid ULID"))
	require.False(t, bag.HasAny())
}
