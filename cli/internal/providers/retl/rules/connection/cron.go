// Package connection holds local validation helpers for rETL connection specs.
//
// The public rETL API stores a cron expression verbatim: it never parses it and
// never enforces a frequency floor, so a schedule that can never run - or that
// would run every three minutes - is accepted by the server and only discovered
// once the connection misbehaves. CheckCron closes that gap locally.
//
// # Why there is a parser here instead of a dependency
//
// Because the dependency was tried and it did not pay. robfig/cron/v3 was
// swapped in for the grammar and measured against this file: 412 code lines
// became 408. A library replaces parsing, which is the cheap half; the analysis
// further down, the part with no library answer, is untouched by it.
//
// What ate the saving is that robfig is more permissive than this check can
// afford, so every difference has to be fenced before an expression reaches it.
// It reads "?" as "*" in every field rather than only the two day fields, so
// "? 1 7 1 5" silently widens to an every-minute schedule. It drops an empty
// list entry, so "0,,5" parses as "0,5". It strips a CRON_TZ= or TZ= prefix and
// evaluates in that zone. It caps day-of-week at 6, so "0 0 * * 7" stops
// parsing and the field has to be folded - folded and not clamped, because
// "1-7/4" selects {1,5} and must not gain Sunday while "1-7/3" selects {1,4,0}
// and must. And it reports one error for the whole expression, which cannot say
// which field was at fault, so the rule below - that malformed text outranks an
// unsupported construct - needs every field parsed separately anyway.
//
// One robfig behaviour is not fenced off but adopted: its rule for what counts
// as a wildcard, which decides whether the two day fields are ANDed or ORed. A
// field is a wildcard when any element is one, and a step wider than 1 drops
// the flag - "*/1" is a wildcard, "*/2" is not - where Vixie keys those
// semantics off the literal "*" alone. robfig's is the reading that matters,
// because robfig is what fires the syncs: rudder-sources schedules through
// cron.ParseStandard, and the expression reaches it verbatim - the public API
// stores it without parsing it. So "0,57 0,23 */2 * MON" is too frequent here,
// which is what robfig does with it: from 2026-01-01 it fires 2026-01-11 23:57
// and again 2026-01-12 00:00, three minutes later. Adopting the rule whole,
// rather than the part of it that is easy to spot in the text, is what keeps
// "3,*", "*,3" and "*/1" the same schedules here as they are there.
//
// So the parser stays - not because a library could not do it, but because the
// fences cost what the parser costs, and this way nothing has to be pinned to a
// package that has not moved since 2019 to keep behaving.
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
//	L / W / # / ? on numeric day values        0 0 L * *, 0 0 5#3   unsupported dialect
//	six or seven fields (seconds, year)        0 0 0 * * *          unsupported dialect
//	CRON_TZ= or TZ= prefix                     CRON_TZ=UTC ...      unsupported dialect
//
// The day extensions are recognised only where Quartz writes them: "#" and
// "nL" in day-of-week, "W", "LW" and "L-n" in day-of-month, bare "L" and "?" in
// either - and only on a day number inside the range that construct allows.
// Everything else keeps its ordinary reading, so "0 0 WED * *", "0 0 * * 9#3",
// "0 0 9#3 * *" and "0 0 * * 5W" stay malformed rather than being excused as
// extensions, as do the name spellings Quartz itself would accept, "0 0 * *
// FRI#2" and "0 0 * * FRIL".
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

	// A syntax check, not a product range: "1969" is a dialect we decline to
	// analyse, "garbage" is malformed.
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
		// The directive is what gets reported, but not before the expression it
		// prefixes has been checked. Malformed text outranks an unsupported
		// construct, and a prefix must not be able to shield it.
		if failure := malformedBody(fields[1:]); failure != nil {
			return nil, failure
		}

		return nil, unsupportedCron("timezone directives such as %q are not supported; expressions are evaluated in UTC", fields[0])
	}

	return parseSchedule(fields)
}

// malformedBody returns the error the expression after a directive carries, or
// nil when it is merely another form we decline to analyse.
func malformedBody(fields []string) *CronCheckResult {
	if len(fields) == 0 {
		return invalidCron("timezone directive is not followed by a cron expression")
	}

	if _, failure := parseSchedule(fields); failure != nil && failure.Status == CronInvalid {
		return failure
	}

	return nil
}

func parseSchedule(fields []string) (*cronSchedule, *CronCheckResult) {
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

// classifyExtendedForm decides whether a six or seven field expression really is
// the seconds (and year) dialect, or just text that happens to have six words.
// Only the former earns a warning; garbage stays invalid.
func classifyExtendedForm(fields []string) *CronCheckResult {
	if _, _, failure := secondsField.parse(fields[0]); failure != nil {
		return failure
	}

	_, failure := parseFields(fields[1:6])
	if failure != nil && failure.Status == CronInvalid {
		return failure
	}

	// The year is checked even when the middle five already produced a warning,
	// so a recognised extension cannot carry a malformed year past the check.
	if len(fields) == 7 {
		if _, _, yearFailure := yearField.parse(fields[6]); yearFailure != nil {
			return yearFailure
		}
	}

	if failure != nil {
		return failure
	}

	if len(fields) == 6 {
		return unsupportedCron("six-field expressions (leading seconds field) are not supported; use the five-field %s form", cronFieldLayout)
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

// anyDayField marks an extension Quartz writes in either day field.
const anyDayField = -1

// quartzExtensions are the Quartz-dialect constructs this package can name but
// cannot analyse. Each pattern matches a whole element rather than a marker
// character, which is what keeps "HELLO" and "WED" malformed instead of being
// mistaken for the L and W extensions.
//
// Each entry also records the field Quartz allows it in and the range its day
// number may take, and both matter: "#" is a day-of-week construct, so "9#3"
// checked against day-of-month would pass a weekday that does not exist, and an
// "L-n" offset runs 0-30 rather than over the day numbers 1-31. A number
// outside its own range is malformed in every dialect, so letting the shape
// excuse it would downgrade an error to a warning.
var quartzExtensions = []struct {
	pattern     *regexp.Regexp
	field       int
	min         int
	max         int
	description string
}{
	{pattern: regexp.MustCompile(`^(\d{1,2})#[1-5]$`), field: dowFieldIndex, min: 0, max: 7, description: `"#" (nth weekday of the month)`},
	{pattern: regexp.MustCompile(`^(\d{1,2})L$`), field: dowFieldIndex, min: 0, max: 7, description: `"L" (last given weekday of the month)`},
	{pattern: regexp.MustCompile(`^L-(\d{1,2})$`), field: domFieldIndex, min: 0, max: 30, description: `"L-n" (offset from the last day)`},
	{pattern: regexp.MustCompile(`^(\d{1,2})W$`), field: domFieldIndex, min: 1, max: 31, description: `"W" (nearest weekday)`},
	{pattern: regexp.MustCompile(`^LW$`), field: domFieldIndex, description: `"LW" (last weekday of the month)`},
	{pattern: regexp.MustCompile(`^L$`), field: anyDayField, description: `"L" (last day)`},
	{pattern: regexp.MustCompile(`^\?$`), field: anyDayField, description: `"?" (no specific value)`},
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
		description, matched := matchExtension(index, element)
		if !matched {
			if _, _, failure := field.parse(element); failure != nil {
				return nil
			}
			continue
		}

		// Only the first match is reported, and only it is formatted: building
		// a message per element would be quadratic in the length of the field.
		if found == nil {
			found = unsupportedCron("%s field %q: the %s extension is not supported", field.name, raw, description)
		}
	}

	return found
}

// matchExtension names the Quartz extension element spells in the day field at
// index, or reports no match - which leaves the element to the parser, and so
// to a malformed verdict. An extension written in the wrong day field, or
// carrying a day number outside its own range, is exactly that: malformed.
func matchExtension(index int, element string) (string, bool) {
	upper := strings.ToUpper(element)
	for _, extension := range quartzExtensions {
		if extension.field != anyDayField && extension.field != index {
			continue
		}

		match := extension.pattern.FindStringSubmatch(upper)
		if match == nil {
			continue
		}
		if len(match) > 1 && !inRange(match[1], extension.min, extension.max) {
			continue
		}

		return extension.description, true
	}

	return "", false
}

// inRange reports whether a captured day number is legal for its extension. The
// patterns match at most two digits, so the capture is always a number and only
// its range is ever in question.
func inRange(number string, min, max int) bool {
	value, _ := strconv.Atoi(number)

	return value >= min && value <= max
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

	if spec == "*" {
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
