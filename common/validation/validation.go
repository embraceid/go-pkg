package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Code string

const (
	RequiredCode  Code = "required"
	MinLengthCode Code = "min_length"
	MaxLengthCode Code = "max_length"
	MinValueCode  Code = "min_value"
	MaxValueCode  Code = "max_value"
	PatternCode   Code = "pattern"
	OneOfCode     Code = "one_of"
	ULIDCode      Code = "ulid"
)

type Param struct {
	Key   string
	Value string
}

type Failure struct {
	Code    Code
	Field   string
	Message string
	Params  []Param
}

type Bag struct {
	failures []Failure
}

var ulidPattern = regexp.MustCompile(`^[0-7][0-9A-HJKMNP-TV-Z]{25}$`)

func NewBag() *Bag {
	return &Bag{failures: []Failure{}}
}

func (b *Bag) Add(code Code, field, message string, params ...Param) {
	if b == nil {
		return
	}

	failureParams := make([]Param, len(params))
	copy(failureParams, params)

	b.failures = append(b.failures, Failure{
		Code:    code,
		Field:   field,
		Message: message,
		Params:  failureParams,
	})
}

func (b *Bag) HasAny() bool {
	return b != nil && len(b.failures) > 0
}

func (b *Bag) IsEmpty() bool {
	return !b.HasAny()
}

func (b *Bag) Len() int {
	if b == nil {
		return 0
	}

	return len(b.failures)
}

func (b *Bag) Failures() []Failure {
	if b == nil || len(b.failures) == 0 {
		return []Failure{}
	}

	failures := make([]Failure, len(b.failures))
	for i, failure := range b.failures {
		failures[i] = failure
		failures[i].Params = append([]Param(nil), failure.Params...)
	}

	return failures
}

func (b *Bag) RequiredString(field, value, message string) bool {
	if strings.TrimSpace(value) != "" {
		return true
	}

	b.Add(RequiredCode, field, message)
	return false
}

func (b *Bag) MinStringLength(field, value string, min int, message string) bool {
	if utf8.RuneCountInString(value) >= min {
		return true
	}

	b.Add(MinLengthCode, field, message, Param{Key: "min", Value: fmt.Sprint(min)})
	return false
}

func (b *Bag) MaxStringLength(field, value string, max int, message string) bool {
	if utf8.RuneCountInString(value) <= max {
		return true
	}

	b.Add(MaxLengthCode, field, message, Param{Key: "max", Value: fmt.Sprint(max)})
	return false
}

func (b *Bag) MinInt(field string, value, min int, message string) bool {
	if value >= min {
		return true
	}

	b.Add(MinValueCode, field, message, Param{Key: "min", Value: fmt.Sprint(min)})
	return false
}

func (b *Bag) MaxInt(field string, value, max int, message string) bool {
	if value <= max {
		return true
	}

	b.Add(MaxValueCode, field, message, Param{Key: "max", Value: fmt.Sprint(max)})
	return false
}

func (b *Bag) Pattern(field, value string, pattern *regexp.Regexp, message string) bool {
	if pattern != nil && pattern.MatchString(value) {
		return true
	}

	b.Add(PatternCode, field, message)
	return false
}

func (b *Bag) OneOfString(field, value, message string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}

	b.Add(OneOfCode, field, message)
	return false
}

func (b *Bag) ULID(field, value, message string) bool {
	if ulidPattern.MatchString(value) {
		return true
	}

	b.Add(ULIDCode, field, message)
	return false
}
