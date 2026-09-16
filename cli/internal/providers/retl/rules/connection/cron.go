// Package connection holds local validation helpers for rETL connection specs.
//
// The public rETL API stores a cron expression verbatim: it never parses it and
// never enforces a frequency floor, so a schedule that can never run - or that
// would run every three minutes - is accepted by the server and only discovered
// once the connection misbehaves. CheckCron closes that gap locally.
//
// # Why there is a parser here instead of a dependency
//
// Rejecting @every and the Quartz extensions is a pre-pass (detectExtension)
// that would sit in front of any parser, and robfig/cron/v3 covers the rest of
// the grammar - but parsing is the cheap half. The frequency proof needs the
// per-field masks and, for Vixie's day semantics, the star flags. robfig does
// expose the masks, but only on the concrete *cron.SpecSchedule behind its
// Schedule interface, and it packs the star flag into an unexported starBit
// that would have to be hardcoded. Its Next() also gives up after roughly five
// years, so it cannot prove that "0 0 30 2 *" never fires. Owning the parser is
// a judgement call rather than a forced one; what tips it is that the analysis,
// not the grammar, is the part with no library answer.
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
//	day-of-week 7 as Sunday                    0 0 * * 7            supported
//	@hourly @daily @midnight @weekly           @daily               supported
//	@monthly @yearly @annually                 @monthly             supported
//	wrapping range                             22-2 * * * *         invalid
//	wrong field count, bad value, bad step     70 * * * *           invalid
//	duration shortcuts                         @every 1h            invalid
//	unknown descriptor                         @fortnightly         invalid
//	date that no calendar day matches          0 0 30 2 *           invalid
//	@reboot                                    @reboot              invalid
//	L / W / # / ? extensions                   0 0 L * *            unsupported dialect
//	six or seven fields (seconds, year)        0 0 0 * * *          unsupported dialect
//	CRON_TZ= or TZ= prefix                     CRON_TZ=UTC ...      unsupported dialect
//
// "@every" and "@reboot" are invalid rather than unsupported on purpose.
// Unsupported means "the product may well run this, we just cannot check it
// here", and a warning lets the apply go through. rETL schedules neither a
// duration shortcut nor a reboot, so warning about them would ship a connection
// that never syncs. For the same reason malformed text outranks an unsupported
// construct: an expression carrying both is an error.
//
// # Analysis
//
// Every expression is evaluated in UTC from a fixed origin, so a result never
// depends on the wall clock, the current date or the developer's timezone. The
// five-minute floor is established by reasoning over the parsed field masks
// rather than by sampling occurrences: fire times are never enumerated.
//
// The only loop is over calendar days, and it is bounded by one Gregorian cycle
// (146097 days), which is a proof rather than a sample because the matching
// dates repeat with that cycle. A call walks the cycle at most once, and only
// an impossible date walks all of it: that worst case measures ~2ms on an M3
// Pro, pinned by BenchmarkCheckCronWorstCase. So there is no cutoff and no
// result that needs qualifying: CronInconclusive is part of the contract for
// DEX-829 but nothing here returns it.
package connection

import (
	"fmt"
	"regexp"
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
	// CronInconclusive is reserved for an analysis that cannot reach a verdict;
	// DEX-829 maps it, like CronUnsupportedDialect, to a warning. The analysis
	// below is exact and has no cutoff, so it never returns this today.
	CronInconclusive CronStatus = "inconclusive"
)

// CronCheckResult is the whole contract: a status to branch on, a reason
// written for the spec author, and for CronTooFrequent the first pair of
// consecutive syncs that breaks the floor. Both timestamps are zero for every
// other status.
type CronCheckResult struct {
	Status         CronStatus
	Reason         string
	Occurrence     time.Time
	NextOccurrence time.Time
}

const (
	minGapMinutes   = 5
	minutesPerDay   = 24 * 60
	cronFieldCount  = 5
	cronFieldLayout = "minute hour day-of-month month day-of-week"

	// Quartz only allows its extensions in the two day fields.
	domFieldIndex = 2
	dowFieldIndex = 4

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

	// Day-of-week allows 7 as a second spelling of Sunday; it is folded onto 0
	// once the mask is built.
	cronFields = [cronFieldCount]cronField{
		{name: "minute", min: 0, max: 59},
		{name: "hour", min: 0, max: 23},
		{name: "day-of-month", min: 1, max: 31},
		{name: "month", min: 1, max: 12, names: monthNames},
		{name: "day-of-week", min: 0, max: 7, names: dayNames},
	}

	secondsField = cronField{name: "seconds", min: 0, max: 59}

	// The year bounds are a syntax check - a year is at most four digits - and
	// not a product range, which has never been established. So "garbage" is
	// malformed while "1969" is merely a dialect we do not analyse.
	yearField = cronField{name: "year", min: 0, max: 9999}

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
		return nil, unsupportedCron("timezone directives such as %q are not supported; expressions are evaluated in UTC", fields[0])
	}

	if strings.HasPrefix(fields[0], "@") {
		return parseDescriptor(fields)
	}

	switch len(fields) {
	case cronFieldCount:
		return parseFields(fields)
	case 6, 7:
		return nil, classifyExtendedForm(fields)
	default:
		return nil, invalidCron("expected %d fields (%s), got %d", cronFieldCount, cronFieldLayout, len(fields))
	}
}

func isTimezoneDirective(field string) bool {
	prefix, _, found := strings.Cut(field, "=")
	if !found {
		return false
	}

	upper := strings.ToUpper(prefix)

	return upper == "TZ" || upper == "CRON_TZ"
}

func parseDescriptor(fields []string) (*cronSchedule, *CronCheckResult) {
	descriptor := strings.ToLower(fields[0])

	if expansion, ok := descriptors[descriptor]; ok {
		if len(fields) > 1 {
			return nil, invalidCron("descriptor %q does not take arguments", fields[0])
		}
		return parseFields(strings.Fields(expansion))
	}

	switch descriptor {
	case "@reboot":
		// Invalid for the same reason as @every: rETL has no reboot to fire on,
		// so a warning would ship a connection that never syncs.
		return nil, invalidCron("@reboot is not part of the supported cron grammar; use a five-field expression such as %q", "0 * * * *")
	case "@every":
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

// classifyExtendedForm decides whether a six or seven field expression really is
// the seconds (and year) dialect, or just text that happens to have six words.
// Only the former earns a warning; garbage stays invalid.
func classifyExtendedForm(fields []string) *CronCheckResult {
	if _, _, failure := secondsField.parse(fields[0]); failure != nil {
		return failure
	}

	if _, failure := parseFields(fields[1:6]); failure != nil {
		return failure
	}

	if len(fields) == 6 {
		return unsupportedCron("six-field expressions (leading seconds field) are not supported; use the five-field %s form", cronFieldLayout)
	}

	if _, _, failure := yearField.parse(fields[6]); failure != nil {
		return failure
	}

	return unsupportedCron("seven-field expressions (leading seconds and trailing year fields) are not supported; use the five-field %s form", cronFieldLayout)
}

func parseFields(fields []string) (*cronSchedule, *CronCheckResult) {
	var (
		masks       [cronFieldCount][]bool
		stars       [cronFieldCount]bool
		unsupported *CronCheckResult
	)
	for i, field := range cronFields {
		values, star, failure := field.parse(fields[i])
		if failure == nil {
			masks[i], stars[i] = values, star
			continue
		}

		// Malformed text outranks an unsupported construct: an expression
		// carrying both is an error, never a warning that lets the apply run.
		extension := detectExtension(i, field, fields[i])
		if extension == nil {
			return nil, failure
		}
		if unsupported == nil {
			unsupported = extension
		}
	}

	if unsupported != nil {
		return nil, unsupported
	}

	schedule := cronSchedule{
		minutes:      selected(masks[0]),
		hours:        selected(masks[1]),
		daysStar:     stars[2],
		weekdaysStar: stars[4],
	}
	copy(schedule.days[:], masks[2])
	copy(schedule.months[:], masks[3])
	copy(schedule.weekdays[:], masks[4][:7])
	if masks[4][7] {
		schedule.weekdays[0] = true
	}

	return &schedule, nil
}

// quartzExtensions are the Quartz-dialect constructs this package can name but
// cannot analyse. Each pattern matches a whole element rather than a marker
// character, which is what keeps "HELLO" and "WED" malformed instead of being
// mistaken for the L and W extensions.
var quartzExtensions = []struct {
	pattern     *regexp.Regexp
	description string
}{
	{pattern: regexp.MustCompile(`^\d{1,2}#[1-5]$`), description: `"#" (nth weekday of the month)`},
	{pattern: regexp.MustCompile(`^(?:L(?:W|-\d{1,2})?|\d{1,2}L)$`), description: `"L" (last day)`},
	{pattern: regexp.MustCompile(`^\d{1,2}W$`), description: `"W" (nearest weekday)`},
	{pattern: regexp.MustCompile(`^\?$`), description: `"?" (no specific value)`},
}

// detectExtension explains a field that failed to parse as an unsupported
// construct, or returns nil to leave it malformed. Quartz confines these forms
// to the two day fields, so the same characters elsewhere stay garbage, and
// every element has to be either a recognised extension or otherwise valid -
// "1,nonsense#2" is malformed, not unsupported.
func detectExtension(index int, field cronField, raw string) *CronCheckResult {
	if index != domFieldIndex && index != dowFieldIndex {
		return nil
	}

	var found *CronCheckResult
	for _, element := range strings.Split(raw, ",") {
		extension := matchExtension(field, raw, element)
		if extension == nil {
			if _, _, failure := field.parse(element); failure != nil {
				return nil
			}
			continue
		}
		if found == nil {
			found = extension
		}
	}

	return found
}

func matchExtension(field cronField, raw, element string) *CronCheckResult {
	upper := strings.ToUpper(element)
	for _, extension := range quartzExtensions {
		if extension.pattern.MatchString(upper) {
			return unsupportedCron("%s field %q: the %s extension is not supported", field.name, raw, extension.description)
		}
	}

	return nil
}

// parse returns the values the field selects, indexed by value, and whether the
// field is a wildcard.
func (f cronField) parse(raw string) ([]bool, bool, *CronCheckResult) {
	var (
		values = make([]bool, f.max+1)
		star   bool
	)

	for _, element := range strings.Split(raw, ",") {
		elementStar, failure := f.parseElement(raw, element, values)
		if failure != nil {
			return nil, false, failure
		}
		star = star || elementStar
	}

	return values, star, nil
}

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

	// A stepped wildcard is still a wildcard, as in Vixie cron, where the day
	// semantics key off the leading "*" rather than the set it expands to.
	if spec == "*" {
		fill(values, f.min, f.max, step)
		return true, nil
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
		Status:         CronTooFrequent,
		Reason:         fmt.Sprintf("consecutive syncs have a %d-minute gap; the minimum supported interval is %d minutes", gap, minGapMinutes),
		Occurrence:     occurrence,
		NextOccurrence: next,
	}
}
