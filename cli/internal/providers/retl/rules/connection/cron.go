// Package connection holds local validation helpers for rETL connection specs.
//
// The public rETL API stores a cron expression verbatim: it never parses it and
// never enforces a frequency floor, so a schedule that can never run - or that
// would run every three minutes - is accepted by the server and only discovered
// once the connection misbehaves. CheckCron closes that gap locally.
//
// # Why there is a parser here instead of a dependency
//
// robfig/cron/v3 is what fires the syncs: rudder-sources schedules through
// cron.ParseStandard and the expression reaches it verbatim, because the public
// API stores it without parsing it. That parser is therefore the authority on
// what this check may accept: everything ParseStandard rejects as bad grammar
// is invalid here, never a warning, because a lenient verdict lets the apply
// through and leaves a connection that never syncs - the failure this package
// exists to prevent. So the grammar below is calibrated against robfig rather
// than against Vixie cron: day-of-week stops at 6, descriptors are matched
// case-sensitively, "?" is a second spelling of "*" in every field, and the
// Quartz day extensions are errors.
//
// Swapping the library in for the grammar was tried and measured on an earlier
// revision of this file, and it came out even: a library replaces parsing,
// which is the cheap half, and leaves the analysis below - the part with no
// library answer - untouched. What the parser here still buys is a message that
// names the field at fault, where robfig reports one error for the whole
// expression.
//
// Two readings are deliberately not robfig's. It drops an empty list entry, so
// "0,,5" runs as "0,5"; that stays an error here, because it is a typo with a
// silent reading. And it strips a CRON_TZ= or TZ= prefix and evaluates in that
// zone, which this package cannot check in UTC, so the prefix is reported as an
// unsupported dialect rather than analysed. The zone name is not checked
// either: robfig resolves it against the host's zone database, which is not
// this process's to answer for, so an unknown zone comes back as the same
// warning rather than as the error robfig would raise.
//
// One robfig behaviour is adopted whole: its rule for what counts as a
// wildcard, which decides whether the two day fields are ANDed or ORed. A field
// is a wildcard when any element is one, and a step wider than 1 drops the flag
// - "*/1" is a wildcard, "*/2" is not - where Vixie keys those semantics off
// the literal "*" alone. So "0,57 0,23 */2 * MON" is too frequent here, which
// is what robfig does with it: from 2026-01-01 it fires 2026-01-11 23:57 and
// again 2026-01-12 00:00, three minutes later. Adopting the rule whole, rather
// than the part of it that is easy to spot in the text, is what keeps "3,*",
// "*,3" and "*/1" the same schedules here as they are there.
//
// # Compatibility matrix
//
//	Construct                                  Example              Result
//	-----------------------------------------  -------------------  -------------------
//	five fields, minute hour dom month dow     0 6 * * *            supported
//	wildcard, lists, ranges                    0,15 1-5 * * *       supported
//	steps on a wildcard or a range             */10, 0-30/10        supported
//	steps from a value (open ended)            15/20 (= 15-59/20)   supported
//	month and day names, name ranges           JAN, mon-fri         supported
//	"?" as a second spelling of "*"            0 0 ? * MON          supported
//	@hourly @daily @midnight @weekly           @daily               supported
//	@monthly @yearly @annually                 @monthly             supported
//	descriptor in any other case               @DAILY               invalid
//	day-of-week 7                              0 0 * * 7            invalid
//	wrapping range                             22-2 * * * *         invalid
//	wrong field count, bad value, bad step     70 * * * *           invalid
//	six or seven fields (seconds, year)        0 0 0 * * *          invalid
//	L / LW / L-n / nW / nL / n#m               0 0 L * *, 0 0 5#3   invalid
//	duration shortcuts                         @every 1h            invalid
//	unknown descriptor                         @fortnightly         invalid
//	@reboot                                    @reboot              invalid
//	date that no calendar day matches          0 0 30 2 *           invalid
//	CRON_TZ= or TZ= prefix                     CRON_TZ=UTC ...      unsupported dialect
//
// The Quartz day extensions are errors and not warnings because ParseStandard
// cannot read any of them - "0 0 L * *" fails there, so a warning would ship a
// connection that never syncs. Day-of-week 7 is the same case: Vixie's second
// spelling of Sunday, which robfig does not take. "?" is the one Quartz form
// that survives, because robfig reads it as "*" in every field rather than only
// in the two day fields Quartz allows it in. It is parsed that way here, so the
// schedule behind it is analysed rather than excused: "? 1 7 1 5" is an
// every-minute schedule and is reported as one.
//
// "@every" and "@reboot" are invalid rather than unsupported on purpose.
// Unsupported means "the product may well run this, we just cannot check it
// here", and a warning lets the apply go through. rETL schedules neither a
// duration shortcut nor a reboot, so warning about them would ship a schedule
// this package has never examined. For the same reason a timezone directive
// does not shield the expression it prefixes: malformed text outranks an
// unsupported construct, and an expression carrying both is an error.
//
// # Analysis
//
// Every expression is evaluated in UTC from a fixed origin, so a result never
// depends on the wall clock, the current date or the developer's timezone. The
// five-minute floor is established by reasoning over the parsed field masks
// rather than by sampling occurrences: fire times are never enumerated.
//
// The only loop is over calendar days, and it is bounded by one Gregorian cycle
// (146097 days), which makes the scan a proof rather than a sample because the
// matching dates repeat with that cycle. Walking a whole cycle is neither rare
// nor a sign of a bad expression: any schedule whose matching days are never
// calendar-adjacent walks all of it before returning valid, "0,57 0,23 29 2 *"
// among them. A call can walk the cycle twice, once to find the first matching
// day and once to look for an adjacent pair. BenchmarkCheckCron pins both
// shapes, and the cost is a few milliseconds - which is why there is no cutoff
// and no result that has to be qualified.
package connection

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CronStatus string

const (
	CronValid              CronStatus = "valid"
	CronInvalid            CronStatus = "invalid"
	CronTooFrequent        CronStatus = "too_frequent"
	CronUnsupportedDialect CronStatus = "unsupported_dialect"
)

// CronCheckResult is the whole contract: a status to branch on, a reason
// written for the spec author, and for CronTooFrequent the first pair of
// consecutive syncs that breaks the floor. Both timestamps are zero for every
// other status, which is why they are named for the violation rather than for
// the schedule.
type CronCheckResult struct {
	Status            CronStatus
	Reason            string
	ViolatingSync     time.Time
	NextViolatingSync time.Time
}

const (
	minGapMinutes   = 5
	minutesPerDay   = 24 * 60
	cronFieldCount  = 5
	cronFieldLayout = "minute hour day-of-month month day-of-week"

	// gregorianCycleDays is the exact repeat period of the Gregorian calendar
	// (400 years) and a multiple of 7, so weekday alignment repeats with it.
	// Scanning one cycle is therefore a proof, not a sample: it settles both
	// "this date never occurs" and the true minimum gap between matching days.
	gregorianCycleDays = 146097
)

// cronOrigin is where every scan starts. Pinned so results are reproducible and
// independent of when validation runs.
var cronOrigin = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// CheckCron classifies expression against the cron grammar documented on this
// package and the five-minute minimum sync interval, in UTC.
func CheckCron(expression string) CronCheckResult {
	return checkCronFrom(expression, cronOrigin)
}

// checkCronFrom exists so tests can move the reference date; origin is
// normalised to the start of its UTC day.
func checkCronFrom(expression string, origin time.Time) CronCheckResult {
	schedule, failure := parseCron(expression)
	if failure != nil {
		return *failure
	}

	return schedule.check(origin)
}

// Parse failures are returned as the result they classify to, so a severity is
// never inferred back out of an error string.
func invalidCron(format string, args ...any) *CronCheckResult {
	return &CronCheckResult{Status: CronInvalid, Reason: fmt.Sprintf(format, args...)}
}

func unsupportedCron(format string, args ...any) *CronCheckResult {
	return &CronCheckResult{Status: CronUnsupportedDialect, Reason: fmt.Sprintf(format, args...)}
}

type cronField struct {
	name  string
	min   int
	max   int
	names map[string]int
}

var (
	monthNames = map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
	dayNames = map[string]int{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
	}

	// Day-of-week stops at 6: robfig does not take 7 as a second spelling of
	// Sunday, so a schedule written that way never runs.
	cronFields = [cronFieldCount]cronField{
		{name: "minute", min: 0, max: 59},
		{name: "hour", min: 0, max: 23},
		{name: "day-of-month", min: 1, max: 31},
		{name: "month", min: 1, max: 12, names: monthNames},
		{name: "day-of-week", min: 0, max: 6, names: dayNames},
	}

	descriptors = map[string]string{
		"@yearly":   "0 0 1 1 *",
		"@annually": "0 0 1 1 *",
		"@monthly":  "0 0 1 * *",
		"@weekly":   "0 0 * * 0",
		"@daily":    "0 0 * * *",
		"@midnight": "0 0 * * *",
		"@hourly":   "0 * * * *",
	}
)

// cronSchedule is the expression reduced to the sets it selects. minutes and
// hours are ascending; daysStar and weekdaysStar record whether the field was
// written as a wildcard, which decides the day matching semantics.
type cronSchedule struct {
	minutes      []int
	hours        []int
	days         [32]bool
	months       [13]bool
	weekdays     [7]bool
	daysStar     bool
	weekdaysStar bool
}

func parseCron(expression string) (*cronSchedule, *CronCheckResult) {
	fields := strings.Fields(expression)
	if len(fields) == 0 {
		return nil, invalidCron("cron expression is empty")
	}

	if isTimezoneDirective(fields[0]) {
		body := fields[1:]
		if len(body) == 0 {
			return nil, invalidCron("timezone directive is not followed by a cron expression")
		}

		// The directive is what gets reported, but not before the expression it
		// prefixes has been checked: a prefix must not shield malformed text
		// behind a warning that lets the apply through.
		if _, failure := parseSchedule(body); failure != nil {
			return nil, failure
		}

		return nil, unsupportedCron("timezone directives such as %q are not supported; expressions are evaluated in UTC", fields[0])
	}

	return parseSchedule(fields)
}

func parseSchedule(fields []string) (*cronSchedule, *CronCheckResult) {
	if strings.HasPrefix(fields[0], "@") {
		return parseDescriptor(fields)
	}

	// The seconds and year dialects are errors rather than warnings because
	// cron.ParseStandard takes exactly five fields: a six-field expression
	// never runs, so naming the extra field is the actionable message.
	switch len(fields) {
	case cronFieldCount:
		return parseFields(fields)
	case 6:
		return nil, invalidCron("six-field expressions (leading seconds field) are not supported; use the five-field %s form", cronFieldLayout)
	case 7:
		return nil, invalidCron("seven-field expressions (leading seconds and trailing year fields) are not supported; use the five-field %s form", cronFieldLayout)
	default:
		return nil, invalidCron("expected %d fields (%s), got %d", cronFieldCount, cronFieldLayout, len(fields))
	}
}

// isTimezoneDirective matches only the two spellings robfig strips, and matches
// them case-sensitively as robfig does: "tz=UTC 0 0 * * *" is a six-field
// expression there, not a prefixed one.
func isTimezoneDirective(field string) bool {
	prefix, _, found := strings.Cut(field, "=")

	return found && (prefix == "TZ" || prefix == "CRON_TZ")
}

// parseDescriptor matches a descriptor exactly as written, because robfig
// switches on the raw string: "@DAILY" is an unrecognised descriptor there, so
// accepting it here would ship a connection that never syncs.
func parseDescriptor(fields []string) (*cronSchedule, *CronCheckResult) {
	descriptor := fields[0]

	if expansion, ok := descriptors[descriptor]; ok {
		if len(fields) > 1 {
			return nil, invalidCron("descriptor %q does not take arguments", fields[0])
		}
		return parseFields(strings.Fields(expansion))
	}

	// @every earns a case of its own because "use */15 * * * *" is actionable.
	// @reboot does not: falling through to the unknown descriptor below already
	// rejects it, with the same status and the supported set spelled out.
	if descriptor == "@every" {
		return nil, invalidCron(
			"duration shortcuts such as %q are not part of the supported cron grammar; use a five-field expression such as %q",
			strings.Join(fields, " "), "*/15 * * * *",
		)
	}

	return nil, invalidCron(
		"unknown descriptor %q; supported descriptors are @hourly, @daily, @midnight, @weekly, @monthly, @yearly and @annually",
		fields[0],
	)
}

func parseFields(fields []string) (*cronSchedule, *CronCheckResult) {
	var (
		masks [cronFieldCount][]bool
		stars [cronFieldCount]bool
	)
	for i, field := range cronFields {
		values, star, failure := field.parse(fields[i])
		if failure != nil {
			return nil, failure
		}
		masks[i], stars[i] = values, star
	}

	schedule := cronSchedule{
		minutes:      selected(masks[0]),
		hours:        selected(masks[1]),
		daysStar:     stars[2],
		weekdaysStar: stars[4],
	}
	copy(schedule.days[:], masks[2])
	copy(schedule.months[:], masks[3])
	copy(schedule.weekdays[:], masks[4])

	return &schedule, nil
}

// parse returns the values the field selects, indexed by value, and whether the
// field is a wildcard.
func (f cronField) parse(raw string) ([]bool, bool, *CronCheckResult) {
	var (
		values = make([]bool, f.max+1)
		star   bool
	)

	// A field is a wildcard when any of its elements is one, which is robfig's
	// rule and so the one the schedules actually run by.
	for _, element := range strings.Split(raw, ",") {
		wildcard, failure := f.parseElement(raw, element, values)
		if failure != nil {
			return nil, false, failure
		}
		star = star || wildcard
	}

	return values, star, nil
}

// parseElement selects the element's values and reports whether it is a
// wildcard. The wildcard flag only decides the day semantics - it never changes
// which values match.
func (f cronField) parseElement(raw, element string, values []bool) (bool, *CronCheckResult) {
	var (
		spec    = element
		step    = 1
		stepped = false
	)

	if base, rawStep, found := strings.Cut(element, "/"); found {
		parsed, err := strconv.Atoi(rawStep)
		if err != nil || parsed < 1 {
			return false, invalidCron("%s field %q: step must be a positive integer", f.name, raw)
		}
		spec, step, stepped = base, parsed, true
	}

	// robfig reads "?" as "*" in every field, not only in the two day fields
	// Quartz allows it in, so the schedule behind it is analysed rather than
	// excused: "? 1 7 1 5" is an every-minute schedule.
	if spec == "*" || spec == "?" {
		fill(values, f.min, f.max, step)

		// robfig keeps the wildcard flag for a step of 1 and drops it for a
		// wider one, so "*/1" is a wildcard and "*/2" is not.
		return step == 1, nil
	}

	low, high, failure := f.parseRange(raw, spec, stepped)
	if failure != nil {
		return false, failure
	}
	fill(values, low, high, step)

	return false, nil
}

func (f cronField) parseRange(raw, spec string, stepped bool) (int, int, *CronCheckResult) {
	start, end, isRange := strings.Cut(spec, "-")

	low, failure := f.parseValue(raw, start)
	if failure != nil {
		return 0, 0, failure
	}

	if !isRange {
		// A step without a range runs to the end of the field, as in "15/20".
		if stepped {
			return low, f.max, nil
		}
		return low, low, nil
	}

	high, failure := f.parseValue(raw, end)
	if failure != nil {
		return 0, 0, failure
	}
	if high < low {
		return 0, 0, invalidCron("%s field %q: range start %d is greater than range end %d", f.name, raw, low, high)
	}

	return low, high, nil
}

func (f cronField) parseValue(raw, token string) (int, *CronCheckResult) {
	if token == "" {
		return 0, invalidCron("%s field %q: value must not be empty", f.name, raw)
	}

	if value, ok := f.names[strings.ToLower(token)]; ok {
		return value, nil
	}

	value, err := strconv.Atoi(token)
	if err != nil {
		if f.names != nil {
			return 0, invalidCron("%s field %q: %q is not a number or %s name", f.name, raw, token, f.name)
		}
		return 0, invalidCron("%s field %q: %q is not a number", f.name, raw, token)
	}
	if value < f.min || value > f.max {
		return 0, invalidCron("%s field %q: %d is out of range %d-%d", f.name, raw, value, f.min, f.max)
	}

	return value, nil
}

// step is whatever positive integer the author typed, so the loop stops on the
// remaining headroom rather than on value+step, which would overflow into a
// negative index for a step near math.MaxInt.
func fill(values []bool, low, high, step int) {
	for value := low; value <= high; value += step {
		values[value] = true
		if step > high-value {
			return
		}
	}
}

func selected(values []bool) []int {
	var out []int
	for value, ok := range values {
		if ok {
			out = append(out, value)
		}
	}

	return out
}

// check proves or disproves the five-minute floor from the field masks alone.
// Two consecutive syncs can only be close together in three places: inside one
// hour, across the join between two selected hours, or across midnight. The
// minimum over those three is the minimum over the whole schedule, so no
// occurrence is ever enumerated.
func (s *cronSchedule) check(origin time.Time) CronCheckResult {
	firstDay, ok := s.nextDay(origin)
	if !ok {
		return CronCheckResult{
			Status: CronInvalid,
			Reason: "schedule never occurs: no calendar date matches the day-of-month, month and day-of-week fields",
		}
	}

	var (
		firstMinute = s.minutes[0]
		lastMinute  = s.minutes[len(s.minutes)-1]
		firstHour   = s.hours[0]
		lastHour    = s.hours[len(s.hours)-1]
	)

	// Checked in chronological order, so the pair reported is the first one.
	for i := 1; i < len(s.minutes); i++ {
		if gap := s.minutes[i] - s.minutes[i-1]; gap < minGapMinutes {
			return tooFrequent(gap, at(firstDay, firstHour, s.minutes[i-1]), at(firstDay, firstHour, s.minutes[i]))
		}
	}

	for i := 1; i < len(s.hours); i++ {
		if gap := (s.hours[i]-s.hours[i-1])*60 + firstMinute - lastMinute; gap < minGapMinutes {
			return tooFrequent(gap, at(firstDay, s.hours[i-1], lastMinute), at(firstDay, s.hours[i], firstMinute))
		}
	}

	// Only calendar-adjacent days can break the floor across midnight: two days
	// apart already puts at least 1441 minutes between the syncs. So the scan
	// for adjacent days is skipped unless a one-day gap would violate.
	gap := minutesPerDay + (firstHour*60 + firstMinute) - (lastHour*60 + lastMinute)
	if gap >= minGapMinutes {
		return CronCheckResult{Status: CronValid}
	}

	day, nextDay, adjacent := s.adjacentDays(firstDay)
	if !adjacent {
		return CronCheckResult{Status: CronValid}
	}

	return tooFrequent(gap, at(day, lastHour, lastMinute), at(nextDay, firstHour, firstMinute))
}

// nextDay returns the first day at or after from whose date matches. A full
// Gregorian cycle without a match proves the schedule never occurs.
func (s *cronSchedule) nextDay(from time.Time) (time.Time, bool) {
	// A UTC day is exactly 24 hours, so truncation lands on midnight.
	day := from.UTC().Truncate(24 * time.Hour)
	for i := 0; i < gregorianCycleDays; i++ {
		if s.matchesDay(day) {
			return day, true
		}
		day = nextUTCDay(day)
	}

	return time.Time{}, false
}

// adjacentDays returns the first pair of calendar-adjacent matching days at or
// after from. One cycle is exhaustive: the matching dates repeat with it.
func (s *cronSchedule) adjacentDays(from time.Time) (time.Time, time.Time, bool) {
	var (
		day     = from
		matches = s.matchesDay(day)
	)
	for i := 0; i < gregorianCycleDays; i++ {
		var (
			next        = nextUTCDay(day)
			nextMatches = s.matchesDay(next)
		)
		if matches && nextMatches {
			return day, next, true
		}
		day, matches = next, nextMatches
	}

	return time.Time{}, time.Time{}, false
}

// matchesDay applies Vixie's day semantics: when both day fields are restricted
// a date matches if either field matches, and a wildcard in one field leaves the
// other in charge.
func (s *cronSchedule) matchesDay(day time.Time) bool {
	if !s.months[int(day.Month())] {
		return false
	}

	var (
		dayMatches     = s.days[day.Day()]
		weekdayMatches = s.weekdays[int(day.Weekday())]
	)
	if s.daysStar || s.weekdaysStar {
		return dayMatches && weekdayMatches
	}

	return dayMatches || weekdayMatches
}

// UTC has no daylight saving, so a day is always exactly 24 hours - which keeps
// the scan cheap enough to walk a 400-year cycle.
func nextUTCDay(day time.Time) time.Time {
	return day.Add(24 * time.Hour)
}

func at(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.UTC)
}

func tooFrequent(gap int, occurrence, next time.Time) CronCheckResult {
	return CronCheckResult{
		Status:            CronTooFrequent,
		Reason:            fmt.Sprintf("consecutive syncs have a %d-minute gap; the minimum supported interval is %d minutes", gap, minGapMinutes),
		ViolatingSync:     occurrence,
		NextViolatingSync: next,
	}
}
