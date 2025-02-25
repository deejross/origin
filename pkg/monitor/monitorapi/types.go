package monitorapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ObservedUpdateCountAnnotation is an annotation added locally (in the monitor only), that tracks how many updates
	// we've seen to this resource.  This is useful during post-processing for determining if we have a hot resource.
	ObservedUpdateCountAnnotation = "monitor.openshift.io/observed-update-count"

	// ObservedRecreationCountAnnotation is an annotation added locally (in the monitor only), that tracks how many
	// time a resource has been recreated.  The internal cache doesn't remove an entry on delete.
	// This is useful during post-processing for determining if we have a hot resource.
	ObservedRecreationCountAnnotation = "monitor.openshift.io/observed-recreation-count"
)

type EventLevel int

const (
	Info EventLevel = iota
	Warning
	Error
)

func (e EventLevel) String() string {
	switch e {
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	default:
		panic(fmt.Sprintf("did not define event level string for %d", e))
	}
}

func EventLevelFromString(s string) (EventLevel, error) {
	switch s {
	case "Info":
		return Info, nil
	case "Warning":
		return Warning, nil
	case "Error":
		return Error, nil
	default:
		return Error, fmt.Errorf("did not define event level string for %q", s)
	}

}

type Condition struct {
	Level EventLevel

	Locator string
	Message string
}

type EventInterval struct {
	Condition

	From time.Time
	To   time.Time
}

func (i EventInterval) String() string {
	if i.From.Equal(i.To) {
		return fmt.Sprintf("%s.%03d %s %s %s", i.From.Format("Jan 02 15:04:05"), i.From.Nanosecond()/int(time.Millisecond), i.Level.String()[:1], i.Locator, strings.Replace(i.Message, "\n", "\\n", -1))
	}
	duration := i.To.Sub(i.From)
	if duration < time.Second {
		return fmt.Sprintf("%s.%03d - %-5s %s %s %s", i.From.Format("Jan 02 15:04:05"), i.From.Nanosecond()/int(time.Millisecond), strconv.Itoa(int(duration/time.Millisecond))+"ms", i.Level.String()[:1], i.Locator, strings.Replace(i.Message, "\n", "\\n", -1))
	}
	return fmt.Sprintf("%s.%03d - %-5s %s %s %s", i.From.Format("Jan 02 15:04:05"), i.From.Nanosecond()/int(time.Millisecond), strconv.Itoa(int(duration/time.Second))+"s", i.Level.String()[:1], i.Locator, strings.Replace(i.Message, "\n", "\\n", -1))
}

type IntervalFilter func(i EventInterval) bool

type IntervalFilters []IntervalFilter

func (filters IntervalFilters) All(i EventInterval) bool {
	for _, filter := range filters {
		if !filter(i) {
			return false
		}
	}
	return true
}

func (filters IntervalFilters) Any(i EventInterval) bool {
	for _, filter := range filters {
		if filter(i) {
			return true
		}
	}
	return false
}

type Intervals []EventInterval

var _ sort.Interface = Intervals{}

func (intervals Intervals) Less(i, j int) bool {
	switch d := intervals[i].From.Sub(intervals[j].From); {
	case d < 0:
		return true
	case d > 0:
		return false
	}
	switch d := intervals[i].To.Sub(intervals[j].To); {
	case d < 0:
		return true
	case d > 0:
		return false
	}
	return intervals[i].Message < intervals[j].Message
}
func (intervals Intervals) Len() int { return len(intervals) }
func (intervals Intervals) Swap(i, j int) {
	intervals[i], intervals[j] = intervals[j], intervals[i]
}

// Strings returns the result of String() on each included interval.
func (intervals Intervals) Strings() []string {
	if len(intervals) == 0 {
		return []string(nil)
	}
	s := make([]string, 0, len(intervals))
	for _, interval := range intervals {
		s = append(s, interval.String())
	}
	return s
}

// Duration returns the sum of all intervals in the range. If To is less than or
// equal to From, 0 is used instead (use Clamp() if open intervals
// should be not considered instant).
// minDuration is the smallest duration to add.  If a duration is less than the minDuration,
// then the minDuration is used instead.  This is useful for measuring samples.
// For example, consider a case of one second polling for server availability.
// If a sample fails, you don't definitively know whether it was down just after t-1s or just before t.
// On average, it would be 500ms, but a useful minimum in this case could be 1s.
func (intervals Intervals) Duration(minCurrentDuration time.Duration) time.Duration {
	var totalDuration time.Duration
	for _, interval := range intervals {
		currentDuration := interval.To.Sub(interval.From)
		if currentDuration <= 0 {
			totalDuration += 0
		} else if currentDuration < minCurrentDuration {
			totalDuration += minCurrentDuration
		} else {
			totalDuration += currentDuration
		}
	}
	return totalDuration
}

// EventIntervalMatchesFunc is a function for matching eventIntervales
type EventIntervalMatchesFunc func(eventInterval EventInterval) bool

// IsErrorEvent returns true if the eventInterval is an Error
func IsErrorEvent(eventInterval EventInterval) bool {
	return eventInterval.Level == Error
}

// IsWarningEvent returns true if the eventInterval is an Warning
func IsWarningEvent(eventInterval EventInterval) bool {
	return eventInterval.Level == Warning
}

// IsInfoEvent returns true if the eventInterval is an Info
func IsInfoEvent(eventInterval EventInterval) bool {
	return eventInterval.Level == Info
}

func And(filters ...EventIntervalMatchesFunc) EventIntervalMatchesFunc {
	return func(eventInterval EventInterval) bool {
		for _, filter := range filters {
			if !filter(eventInterval) {
				return false
			}
		}
		return true
	}
}

func Or(filters ...EventIntervalMatchesFunc) EventIntervalMatchesFunc {
	return func(eventInterval EventInterval) bool {
		for _, filter := range filters {
			if filter(eventInterval) {
				return true
			}
		}
		return false
	}
}

// Filter returns a copy of intervals with only intervals that match the provided
// function.
func (intervals Intervals) Filter(eventFilterMatches EventIntervalMatchesFunc) Intervals {
	if len(intervals) == 0 {
		return Intervals(nil)
	}
	copied := make(Intervals, 0, len(intervals))
	for _, interval := range intervals {
		if eventFilterMatches(interval) {
			copied = append(copied, interval)
		}
	}
	return copied
}

// Cut creates a copy of intervals where all events (empty To) are
// within [from,to) and all intervals that overlap [from,to) are
// included, but with their from/to fields limited to that range.
func (intervals Intervals) Cut(from, to time.Time) Intervals {
	if len(intervals) == 0 {
		return Intervals(nil)
	}
	copied := make(Intervals, 0, len(intervals))
	for _, interval := range intervals {
		if interval.To.IsZero() {
			if interval.From.IsZero() {
				continue
			}
			if interval.From.Before(from) || !interval.From.Before(to) {
				continue
			}
		} else {
			if interval.To.Before(from) || !interval.From.Before(to) {
				continue
			}
			// limit the interval to the provided range
			if interval.To.After(to) {
				interval.To = to
			}
			if interval.From.Before(from) {
				interval.From = from
			}
		}
		copied = append(copied, interval)
	}
	return copied
}

// CopyAndSort assumes intervals is unsorted and returns a sorted copy of intervals
// for all intervals between from and to.
func (intervals Intervals) CopyAndSort(from, to time.Time) Intervals {
	copied := make(Intervals, 0, len(intervals))

	if from.IsZero() && to.IsZero() {
		for _, e := range intervals {
			copied = append(copied, e)
		}
		sort.Sort(copied)
		return copied
	}

	for _, e := range intervals {
		if !e.From.After(from) {
			continue
		}
		if !to.IsZero() && !e.From.Before(to) {
			continue
		}
		copied = append(copied, e)
	}
	sort.Sort(copied)
	return copied

}

// Slice works on a sorted Intervals list and returns the set of intervals
// that start after from and start before to (if to is set). The zero value will
// return all elements. If intervals is unsorted the result is undefined. This
// runs in O(n).
func (intervals Intervals) Slice(from, to time.Time) Intervals {
	if from.IsZero() && to.IsZero() {
		return intervals
	}

	first := sort.Search(len(intervals), func(i int) bool {
		return intervals[i].From.After(from)
	})
	if first == -1 {
		return nil
	}
	if to.IsZero() {
		return intervals[first:]
	}
	for i := first; i < len(intervals); i++ {
		if intervals[i].From.After(to) {
			return intervals[first:i]
		}
	}
	return intervals[first:]
}

// Clamp sets all zero value From or To fields to from or to.
func (intervals Intervals) Clamp(from, to time.Time) {
	for i := range intervals {
		if intervals[i].From.IsZero() {
			intervals[i].From = from
		}
		if intervals[i].To.IsZero() {
			intervals[i].To = to
		}
	}
}

type InstanceMap map[string]runtime.Object
type ResourcesMap map[string]InstanceMap
