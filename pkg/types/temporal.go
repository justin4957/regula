// Package types provides the core domain types for regulation modeling.
// Ported from lex-sim (Crisp) to Go.
package types

import (
	"time"
)

// Date represents a calendar date without time component.
// Implements comparison via time.Time.
type Date struct {
	Year  int
	Month int // 1-12
	Day   int // 1-31
}

// ToTime converts a Date to a time.Time at midnight UTC.
func (d Date) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// FromTime creates a Date from a time.Time.
func FromTime(t time.Time) Date {
	return Date{
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
	}
}

// Today returns the current date.
func Today() Date {
	return FromTime(time.Now())
}

// Before returns true if d is before other.
func (d Date) Before(other Date) bool {
	return d.ToTime().Before(other.ToTime())
}

// After returns true if d is after other.
func (d Date) After(other Date) bool {
	return d.ToTime().After(other.ToTime())
}

// Equal returns true if d equals other.
func (d Date) Equal(other Date) bool {
	return d.Year == other.Year && d.Month == other.Month && d.Day == other.Day
}

// BeforeOrEqual returns true if d is before or equal to other.
func (d Date) BeforeOrEqual(other Date) bool {
	return d.Before(other) || d.Equal(other)
}

// AfterOrEqual returns true if d is after or equal to other.
func (d Date) AfterOrEqual(other Date) bool {
	return d.After(other) || d.Equal(other)
}

// DateMax is the maximum representable date.
var DateMax = Date{Year: 9999, Month: 12, Day: 31}

// DateMin is the minimum representable date.
var DateMin = Date{Year: 0, Month: 1, Day: 1}

// Timestamp represents a precise point in time.
type Timestamp struct {
	Date   Date
	Hour   int // 0-23
	Minute int // 0-59
	Second int // 0-59
	TZ     Timezone
}

// TimezoneKind represents the type of timezone specification.
type TimezoneKind int

const (
	TimezoneUTC TimezoneKind = iota
	TimezoneOffset
	TimezoneNamed
)

// Timezone represents a timezone.
type Timezone struct {
	Kind    TimezoneKind
	Hours   int    // for Offset
	Minutes int    // for Offset
	Name    string // for Named (e.g., "America/New_York")
}

// UTC returns a UTC timezone.
func UTC() Timezone {
	return Timezone{Kind: TimezoneUTC}
}

// TemporalRange represents a range of time during which something is valid.
type TemporalRange struct {
	EffectiveFrom  Date
	EffectiveUntil *Date // nil = still in force
}

// IsValidAt checks if the temporal range is valid at a given date.
func (r TemporalRange) IsValidAt(date Date) bool {
	if date.Before(r.EffectiveFrom) {
		return false
	}
	if r.EffectiveUntil != nil && date.After(*r.EffectiveUntil) {
		return false
	}
	return true
}

// Overlaps checks if two temporal ranges overlap.
func (r TemporalRange) Overlaps(other TemporalRange) bool {
	r1End := DateMax
	if r.EffectiveUntil != nil {
		r1End = *r.EffectiveUntil
	}
	r2End := DateMax
	if other.EffectiveUntil != nil {
		r2End = *other.EffectiveUntil
	}
	return r.EffectiveFrom.BeforeOrEqual(r2End) && other.EffectiveFrom.BeforeOrEqual(r1End)
}

// IsCurrent returns true if the range is currently in effect with no end date.
func (r TemporalRange) IsCurrent() bool {
	return r.EffectiveUntil == nil && r.IsValidAt(Today())
}

// AmendmentID identifies an amendment to a provision.
type AmendmentID struct {
	ID          string
	Enacted     Date
	Effective   Date   // May differ from Enacted
	Description string
}

// AmendmentRecord contains full details of an amendment including before/after text.
type AmendmentRecord struct {
	ID         AmendmentID
	Provision  ProvisionID // What was amended
	BeforeText string
	AfterText  string
	Rationale  *string
}

// SupersessionRecord records when one provision replaces another.
type SupersessionRecord struct {
	SupersededOn Date
	SupersededBy ProvisionID
	Partial      bool    // True if only partially superseded
	Scope        *string // Description of scope if partial
}

// TemporalValidity tracks the validity of a provision over time.
type TemporalValidity struct {
	Range      TemporalRange
	AmendedBy  []AmendmentID
	Superseded *SupersessionRecord
}

// IsValidAt checks if the validity is active at a given date.
func (v TemporalValidity) IsValidAt(date Date) bool {
	return v.Range.IsValidAt(date) && v.Superseded == nil
}

// IsCurrent returns true if currently valid and not superseded.
func (v TemporalValidity) IsCurrent() bool {
	return v.Range.IsCurrent() && v.Superseded == nil
}

// TemporalQueryKind represents the type of temporal query.
type TemporalQueryKind int

const (
	TemporalQueryAsOf TemporalQueryKind = iota
	TemporalQueryBetween
	TemporalQueryCurrent
	TemporalQueryHistorical
)

// TemporalQuery specifies a point or range for legal research.
type TemporalQuery struct {
	Kind  TemporalQueryKind
	AsOf  *Date // for AsOf
	Start *Date // for Between and Historical
	End   *Date // for Between and Historical (nil for open-ended)
}

// AsOfQuery creates an AsOf temporal query.
func AsOfQuery(date Date) TemporalQuery {
	return TemporalQuery{Kind: TemporalQueryAsOf, AsOf: &date}
}

// BetweenQuery creates a Between temporal query.
func BetweenQuery(start, end Date) TemporalQuery {
	return TemporalQuery{Kind: TemporalQueryBetween, Start: &start, End: &end}
}

// CurrentQuery creates a Current temporal query.
func CurrentQuery() TemporalQuery {
	return TemporalQuery{Kind: TemporalQueryCurrent}
}

// HistoricalQuery creates a Historical temporal query.
func HistoricalQuery(start Date, end *Date) TemporalQuery {
	return TemporalQuery{Kind: TemporalQueryHistorical, Start: &start, End: end}
}

// Matches checks if a temporal validity matches the query.
func (q TemporalQuery) Matches(validity TemporalValidity) bool {
	switch q.Kind {
	case TemporalQueryAsOf:
		if q.AsOf == nil {
			return false
		}
		return validity.Range.IsValidAt(*q.AsOf)

	case TemporalQueryBetween:
		if q.Start == nil || q.End == nil {
			return false
		}
		queryRange := TemporalRange{EffectiveFrom: *q.Start, EffectiveUntil: q.End}
		return validity.Range.Overlaps(queryRange)

	case TemporalQueryCurrent:
		return validity.IsCurrent()

	case TemporalQueryHistorical:
		if q.Start == nil {
			return false
		}
		if q.End == nil {
			return validity.Range.EffectiveFrom.AfterOrEqual(*q.Start)
		}
		queryRange := TemporalRange{EffectiveFrom: *q.Start, EffectiveUntil: q.End}
		return validity.Range.Overlaps(queryRange)

	default:
		return false
	}
}
