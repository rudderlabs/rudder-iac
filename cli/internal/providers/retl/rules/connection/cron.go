// Package connection holds local validation helpers for rETL connection specs.
//
// The public rETL API stores a cron expression verbatim: it never parses it and
// never enforces a frequency floor, so a schedule that can never run - or that
// would run every three minutes - is accepted by the server and only discovered
// once the connection misbehaves. CheckCron closes that gap locally.
//
// # What robfig/cron owns and what this file owns
//
// robfig/cron/v3 owns the grammar: it parses the five standard fields and hands
// back the per-field bitmasks the frequency proof reasons over. It does not
// answer the question this package exists for - whether a schedule is too
// frequent, or whether it fires at all - and it is permissive in three places
// rETL cannot afford to be, so the expression is fenced before it gets there:
//
//   - it reads "?" as a synonym for "*" and silently strips a CRON_TZ= or TZ=
//     prefix. Both are reported here rather than quietly reinterpreted.
//   - it drops an empty list entry, so "0,,5" would parse as "0,5".
//   - it caps day-of-week at 6, while Vixie - and rETL specs - also spell Sunday
//     as 7, so a 7 is folded onto 0 before parsing.
//
// One further difference is deliberate on robfig's side and wrong for this
// check: it clears its wildcard flag for a stepped wildcard, so "*/2" stops
// counting as "*" and the two day fields switch from AND to OR. Vixie keys
// those semantics off the literal "*", so the two flags are read from the
// expression text here instead of from the parsed schedule.
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
//	L / W / # / ? on numeric day values        0 0 L * *, 0 0 5#3   unsupported dialect
//	six or seven fields (seconds, year)        0 0 0 * * *          unsupported dialect
//	CRON_TZ= or TZ= prefix                     CRON_TZ=UTC ...      unsupported dialect
//
// The day extensions are recognised only where Quartz writes them: in the two
// day fields, on a day number inside that field's range. Everything else keeps
// its usual reading, so "0 0 WED * *" and "0 0 * * 9#3" stay malformed rather
// than being excused as extensions - and so do the name spellings Quartz would
// accept, "0 0 * * FRI#2" and "0 0 * * FRIL".
//
// "@every" and "@reboot" are invalid rather than unsupported on purpose.
// Unsupported means "the product may well run this, we just cannot check it
// here", and a warning lets the apply go through. rETL schedules neither a
// duration shortcut nor a reboot, so warning about them would ship a connection
// that never syncs. For the same reason malformed text outranks an unsupported
// construct: an expression carrying both is an error. That ordering is why each
// field is parsed on its own below - robfig reports one error for the whole
// expression, which cannot say which field was at fault.
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
// matching dates repeat with that cycle. Walking a whole cycle is not rare and
// not limited to a bad expression: any schedule whose matching days are never
// calendar-adjacent walks all of it before returning valid, "0,57 0,23 29 2 *"
// among them. A call can walk the cycle twice, once to find the first matching
// day and once to look for an adjacent pair. BenchmarkCheckCron pins both
// shapes; the cost is a few milliseconds, which is why there is no cutoff and
// no result that has to be qualified.
package connection

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
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

// cronParser is the five-field standard parser. The descriptor option is left
// off deliberately: descriptors are expanded below instead, because robfig's
// set includes @every and @reboot and its error for an unknown one does not
// name the set that is supported.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

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
	name string
	min  int
	max  int
}

var (
	// Day-of-week allows 7 as a second spelling of Sunday, so its range is
	// wider than robfig's; foldSunday reconciles the two.
	cronFields = [cronFieldCount]cronField{
		{name: "minute", min: 0, max: 59},
		{name: "hour", min: 0, max: 23},
		{name: "day-of-month", min: 1, max: 31},
		{name: "month", min: 1, max: 12},
		{name: "day-of-week", min: 0, max: 7},
	}

	// A seconds field carries the same bounds as a minute field, so it probes
	// as one under a different name.
	secondsField = cronField{name: "seconds", min: 0, max: 59}

	// The year is checked for shape - a year is at most four digits - and not
	// for a product range, which has never been established. So "garbage" is
	// malformed while "1969" is merely a dialect we do not analyse.
	yearShape = regexp.MustCompile(`^[\d*][\d*,/-]*$`)

	dayNames = map[string]int{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
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
// hours are ascending; days, months and weekdays stay as robfig's bitmasks.
// daysStar and weekdaysStar record whether the field was written as a wildcard,
// which decides the day matching semantics.
type cronSchedule struct {
	minutes      []int
	hours        []int
	days         uint64
	months       uint64
	weekdays     uint64
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

	if descriptor == "@every" {
		return nil, invalidCron(
			"duration shortcuts such as %q are not part of the supported cron grammar; use a five-field expression such as %q",
			strings.Join(fields, " "), "*/15 * * * *",
		)
	}

	expansion, ok := descriptors[descriptor]
	if !ok {
		return nil, invalidCron(
			"unknown descriptor %q; supported descriptors are @hourly, @daily, @midnight, @weekly, @monthly, @yearly and @annually",
			fields[0],
		)
	}
	if len(fields) > 1 {
		return nil, invalidCron("descriptor %q does not take arguments", fields[0])
	}

	return parseFields(strings.Fields(expansion))
}

// classifyExtendedForm decides whether a six or seven field expression really is
// the seconds (and year) dialect, or just text that happens to have six words.
// Only the former earns a warning; garbage stays invalid.
func classifyExtendedForm(fields []string) *CronCheckResult {
	if failure := secondsField.probe(0, fields[0], fields[0]); failure != nil {
		return failure
	}

	if _, failure := parseFields(fields[1:6]); failure != nil {
		return failure
	}

	if len(fields) == 6 {
		return unsupportedCron("six-field expressions (leading seconds field) are not supported; use the five-field %s form", cronFieldLayout)
	}

	if !yearShape.MatchString(fields[6]) {
		return invalidCron("year field %q: %q is not a number", fields[6], fields[6])
	}

	return unsupportedCron("seven-field expressions (leading seconds and trailing year fields) are not supported; use the five-field %s form", cronFieldLayout)
}

func parseFields(fields []string) (*cronSchedule, *CronCheckResult) {
	var normalised [cronFieldCount]string
	copy(normalised[:], fields)
	normalised[dowFieldIndex] = foldSunday(fields[dowFieldIndex])

	// A malformed field outranks an unsupported construct in another, so every
	// field is examined before either verdict is returned - and the extension
	// check runs even when the field parsed, because robfig accepts "?".
	var unsupported *CronCheckResult
	for i, field := range cronFields {
		var (
			failure   = field.probe(i, fields[i], normalised[i])
			extension = detectExtension(i, field, fields[i])
		)
		if failure != nil && extension == nil {
			return nil, failure
		}
		if extension != nil && unsupported == nil {
			unsupported = extension
		}
	}

	if unsupported != nil {
		return nil, unsupported
	}

	schedule, err := cronParser.Parse(strings.Join(normalised[:], " "))
	spec, ok := schedule.(*cron.SpecSchedule)
	if err != nil || !ok {
		// Unreachable: every field has just parsed on its own, and without the
		// descriptor option a five-field expression is always a spec schedule.
		return nil, invalidCron("%q is not a valid cron expression", strings.Join(fields, " "))
	}

	return &cronSchedule{
		minutes:      selected(spec.Minute, cronFields[0]),
		hours:        selected(spec.Hour, cronFields[1]),
		days:         spec.Dom,
		months:       spec.Month,
		weekdays:     spec.Dow,
		daysStar:     isWildcard(fields[domFieldIndex]),
		weekdaysStar: isWildcard(fields[dowFieldIndex]),
	}, nil
}

// probe parses raw on its own, with wildcards standing in for every other
// field, because robfig reports a single error for the whole expression and
// cannot say which field produced it. normalised carries any Sunday fold; raw
// is what the author wrote and what the message quotes.
func (f cronField) probe(index int, raw, normalised string) *CronCheckResult {
	for _, element := range strings.Split(raw, ",") {
		if element == "" {
			// robfig drops an empty list entry silently, so "0,,5" would parse
			// as "0,5". Vixie rejects it and so does this.
			return invalidCron("%s field %q: value must not be empty", f.name, raw)
		}
		if strings.Contains(element, "?") {
			// robfig reads "?" as "*" in every field. Quartz writes it only in
			// the two day fields, where detectExtension names it as a dialect
			// we decline to analyse; anywhere else it is not grammar at all and
			// must not quietly widen the field to a wildcard.
			return invalidCron("%s field %q: %q is not part of the supported cron grammar", f.name, raw, "?")
		}
	}

	probe := [cronFieldCount]string{"*", "*", "*", "*", "*"}
	probe[index] = normalised
	if _, err := cronParser.Parse(strings.Join(probe[:], " ")); err != nil {
		// robfig wraps strconv's error verbatim. The tail repeats the token and
		// names a standard library function, neither of which tells a spec
		// author anything, so only the sentence robfig wrote is kept.
		reason, _, _ := strings.Cut(err.Error(), ": strconv.")

		return invalidCron("%s field %q: %s", f.name, raw, reason)
	}

	return nil
}

// foldSunday rewrites the day-of-week field so robfig, which stops at 6, sees
// the same set Vixie does. Only a lone value or a range end can be 7, so the
// element keeps its shape with the range clamped to 6 and an explicit "0" is
// appended when the step actually lands on Sunday. Anything malformed is left
// alone for the parser to reject.
func foldSunday(field string) string {
	var (
		elements = strings.Split(field, ",")
		folded   = make([]string, 0, len(elements)+1)
		sunday   bool
	)

	for _, element := range elements {
		spec, rawStep, stepped := strings.Cut(element, "/")
		start, end, isRange := strings.Cut(spec, "-")

		if (isRange && end != "7") || (!isRange && start != "7") {
			folded = append(folded, element)
			continue
		}

		// "7" and "7/n" both select Sunday and nothing else.
		if !isRange {
			sunday = true
			continue
		}

		low, err := dayValue(start)
		step := 1
		if stepped {
			step, err = strconv.Atoi(rawStep)
		}
		if err != nil || step < 1 || low > 7 {
			folded = append(folded, element)
			continue
		}

		if (7-low)%step == 0 {
			sunday = true
		}
		if low <= 6 {
			clamped := start + "-6"
			if stepped {
				clamped += "/" + rawStep
			}
			folded = append(folded, clamped)
		}
	}

	if sunday {
		folded = append(folded, "0")
	}

	return strings.Join(folded, ",")
}

func dayValue(token string) (int, error) {
	if value, ok := dayNames[strings.ToLower(token)]; ok {
		return value, nil
	}

	return strconv.Atoi(token)
}

// isWildcard reports whether the field was written as a wildcard, which is what
// Vixie keys the day semantics off. A stepped wildcard still counts, which is
// where robfig's own flag differs.
func isWildcard(field string) bool {
	for _, element := range strings.Split(field, ",") {
		if strings.HasPrefix(element, "*") {
			return true
		}
	}

	return false
}

// quartzExtensions are the Quartz-dialect constructs this package can name but
// cannot analyse. Each pattern matches a whole element rather than a marker
// character, which is what keeps "HELLO" and "WED" malformed instead of being
// mistaken for the L and W extensions, and captures the day number so it can be
// range-checked.
var quartzExtensions = []struct {
	pattern     *regexp.Regexp
	description string
}{
	{pattern: regexp.MustCompile(`^(\d{1,2})#[1-5]$`), description: `"#" (nth weekday of the month)`},
	{pattern: regexp.MustCompile(`^(?:L(?:W|-(\d{1,2}))?|(\d{1,2})L)$`), description: `"L" (last day)`},
	{pattern: regexp.MustCompile(`^(\d{1,2})W$`), description: `"W" (nearest weekday)`},
	{pattern: regexp.MustCompile(`^\?$`), description: `"?" (no specific value)`},
}

// detectExtension explains a field as an unsupported construct, or returns nil
// to leave it to the parser. Quartz confines these forms to the two day fields,
// so the same characters elsewhere stay garbage, and every element has to be
// either a recognised extension or otherwise valid - "1,nonsense#2" is
// malformed, not unsupported.
func detectExtension(index int, field cronField, raw string) *CronCheckResult {
	if index != domFieldIndex && index != dowFieldIndex {
		return nil
	}

	var found *CronCheckResult
	for _, element := range strings.Split(raw, ",") {
		extension := field.matchExtension(raw, element)
		if extension == nil {
			if failure := field.probe(index, element, element); failure != nil {
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

// matchExtension names the Quartz extension an element spells, or returns nil
// when it is not one - including when it carries a day number outside the
// field's range. "9#3" and "32L" are malformed in every dialect, so excusing
// them as a recognised extension would downgrade an error to a warning and let
// a schedule nothing can run reach the apply.
func (f cronField) matchExtension(raw, element string) *CronCheckResult {
	upper := strings.ToUpper(element)
	for _, extension := range quartzExtensions {
		match := extension.pattern.FindStringSubmatch(upper)
		if match == nil || !f.inRange(match[1:]) {
			continue
		}

		return unsupportedCron("%s field %q: the %s extension is not supported", f.name, raw, extension.description)
	}

	return nil
}

// inRange reports whether every day number a pattern captured is a legal value
// for the field. The patterns match at most two digits, so the capture is
// always a number and only its range is in question.
func (f cronField) inRange(numbers []string) bool {
	for _, number := range numbers {
		if number == "" {
			continue
		}

		value, _ := strconv.Atoi(number)
		if value < f.min || value > f.max {
			return false
		}
	}

	return true
}

func selected(bits uint64, field cronField) []int {
	var out []int
	for value := field.min; value <= field.max; value++ {
		if bits&bit(value) != 0 {
			out = append(out, value)
		}
	}

	return out
}

// bit indexes a robfig field mask. Its wildcard flag lives in the top bit, well
// clear of every value any field can hold, so it never has to be masked off.
func bit(value int) uint64 {
	return 1 << uint(value)
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
	if s.months&bit(int(day.Month())) == 0 {
		return false
	}

	var (
		dayMatches     = s.days&bit(day.Day()) != 0
		weekdayMatches = s.weekdays&bit(int(day.Weekday())) != 0
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
