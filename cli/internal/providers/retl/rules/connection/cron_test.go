package connection

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func utc(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func TestCheckCronFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expression string
		expected   CronCheckResult
	}{
		{
			expression: "*/5 * * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			expression: "5,10 * * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			expression: "0 * * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			expression: "0 0 * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			expression: "* * * * *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 1-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2026, time.January, 1, 0, 0),
				NextOccurrence: utc(2026, time.January, 1, 0, 1),
			},
		},
		{
			expression: "*/3 * * * *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2026, time.January, 1, 0, 0),
				NextOccurrence: utc(2026, time.January, 1, 0, 3),
			},
		},
		{
			// An irregular minute list only breaks the floor at the tail of the
			// list, which an "every N minutes" reading of the field would miss.
			expression: "0,10,20,30,40,50,53 * * * *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2026, time.January, 1, 0, 50),
				NextOccurrence: utc(2026, time.January, 1, 0, 53),
			},
		},
		{
			// The violation is the join between two selected hours, not a gap
			// inside either of them.
			expression: "0,58 0,1 * * *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 2-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2026, time.January, 1, 0, 58),
				NextOccurrence: utc(2026, time.January, 1, 1, 0),
			},
		},
		{
			// The violation only shows up across midnight.
			expression: "0,57 0,23 * * *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2026, time.January, 1, 23, 57),
				NextOccurrence: utc(2026, time.January, 2, 0, 0),
			},
		},
		{
			// Exactly five minutes across midnight is accepted.
			expression: "0,55 0,23 * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			// Leap day: the reference origin is 2026, so the first occurrence is
			// two years out and the violation still has to be reported.
			expression: "0,3 0 29 2 *",
			expected: CronCheckResult{
				Status:         CronTooFrequent,
				Reason:         "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				Occurrence:     utc(2028, time.February, 29, 0, 0),
				NextOccurrence: utc(2028, time.February, 29, 0, 3),
			},
		},
		{
			expression: "0,30 0 29 2 *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			// The day-boundary gap is under five minutes, but no two leap days
			// are ever adjacent. Proving that drives the full 400-year scan.
			expression: "0,57 0,23 29 2 *",
			expected:   CronCheckResult{Status: CronValid},
		},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, CheckCron(test.expression))
		})
	}
}

func TestCheckCronSupportedGrammar(t *testing.T) {
	t.Parallel()

	expressions := []string{
		"0 6 * * *",
		"0 6 * * MON-FRI",
		"0 0 * * mon",
		"0 0 1 JAN *",
		"0 0 1 jan-mar *",
		"0-30/10 * * * *",
		"15/20 * * * *",
		"0,15,30,45 * * * *",
		"0 0 * * 7",
		"0 0 * * 0-7",
		"0 0 29 2 *",
		"@hourly",
		"@daily",
		"@midnight",
		"@weekly",
		"@monthly",
		"@yearly",
		"@annually",
		"@DAILY",
		"  0   0   *   *   *  ",
		// Day-of-month and day-of-week are both restricted, so Vixie's OR
		// semantics apply and every Friday matches alongside the 13th.
		"0 0 13 * FRI",
		// February 30th never exists, but the OR keeps Mondays in February.
		"0 0 30 2 1",
		// A stepped wildcard is still a wildcard, so the day fields are ANDed
		// and only odd-numbered Mondays match - never two days running. Read as
		// OR, Jan 31 and Feb 1 would both match and the three-minute midnight
		// gap would make this too frequent.
		"0,57 0,23 */2 * MON",
		// A step wider than the field selects only its start value. The mask is
		// filled by repeated addition, so a step near math.MaxInt used to
		// overflow into a negative index and panic.
		"1/9223372036854775807 * * * *",
		"0 0 1/9223372036854775807 * *",
	}

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, CronCheckResult{Status: CronValid}, CheckCron(expression))
		})
	}
}

func TestCheckCronInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		reason     string
	}{
		{
			name:       "empty",
			expression: "   ",
			reason:     "cron expression is empty",
		},
		{
			name:       "too few fields",
			expression: "* * *",
			reason:     "expected 5 fields (minute hour day-of-month month day-of-week), got 3",
		},
		{
			name:       "garbage text",
			expression: "jibber jabber nonsense text here",
			reason:     `minute field "jibber": "jibber" is not a number`,
		},
		{
			name:       "six garbage fields stay invalid",
			expression: "a b c d e f",
			reason:     `seconds field "a": "a" is not a number`,
		},
		{
			name:       "seven garbage fields stay invalid",
			expression: "a b c d e f g",
			reason:     `seconds field "a": "a" is not a number`,
		},
		{
			// Garbage carrying an extension marker is still garbage: warning
			// about the L in HELLO would let a broken schedule apply.
			name:       "garbage containing an extension marker",
			expression: "0 0 HELLO * *",
			reason:     `day-of-month field "HELLO": "HELLO" is not a number`,
		},
		{
			name:       "day name in the day-of-month field",
			expression: "0 0 WED * *",
			reason:     `day-of-month field "WED": "WED" is not a number`,
		},
		{
			name:       "extension marker outside the day fields",
			expression: "0 L * * *",
			reason:     `hour field "L": "L" is not a number`,
		},
		{
			// Malformed text outranks an unsupported construct whichever order
			// the two appear in, so an error is never downgraded to a warning.
			name:       "malformed field before an unsupported one",
			expression: "garbage 0 L * *",
			reason:     `minute field "garbage": "garbage" is not a number`,
		},
		{
			name:       "malformed field after an unsupported one",
			expression: "0 0 L * garbage",
			reason:     `day-of-week field "garbage": "garbage" is not a number or day-of-week name`,
		},
		{
			name:       "garbage beside an extension in the same list",
			expression: "0 0 1,nonsense#2 * *",
			reason:     `day-of-month field "1,nonsense#2": "nonsense#2" is not a number`,
		},
		{
			// The year is checked for shape, not for a product range, so text
			// that is not a year at all stays malformed.
			name:       "garbage year",
			expression: "0 0 0 * * * garbage",
			reason:     `year field "garbage": "garbage" is not a number`,
		},
		{
			// rETL has no reboot to fire on, so this is rejected like @every
			// rather than warned about.
			name:       "reboot descriptor",
			expression: "@reboot",
			reason:     `@reboot is not part of the supported cron grammar; use a five-field expression such as "0 * * * *"`,
		},
		{
			name:       "minute out of range",
			expression: "70 * * * *",
			reason:     `minute field "70": 70 is out of range 0-59`,
		},
		{
			name:       "hour out of range",
			expression: "* 25 * * *",
			reason:     `hour field "25": 25 is out of range 0-23`,
		},
		{
			name:       "zero step",
			expression: "*/0 * * * *",
			reason:     `minute field "*/0": step must be a positive integer`,
		},
		{
			name:       "wrapping range",
			expression: "22-2 * * * *",
			reason:     `minute field "22-2": range start 22 is greater than range end 2`,
		},
		{
			name:       "empty list entry",
			expression: "0,,5 * * * *",
			reason:     `minute field "0,,5": value must not be empty`,
		},
		{
			name:       "unknown month name",
			expression: "* * * FOO *",
			reason:     `month field "FOO": "FOO" is not a number or month name`,
		},
		{
			name:       "unknown day name",
			expression: "* * * * FOODAY",
			reason:     `day-of-week field "FOODAY": "FOODAY" is not a number or day-of-week name`,
		},
		{
			// Duration shortcuts are a robfig/cron extension the product does
			// not schedule, so letting them through as a warning would ship a
			// connection that never syncs.
			name:       "duration shortcut",
			expression: "@every 1h",
			reason:     `duration shortcuts such as "@every 1h" are not part of the supported cron grammar; use a five-field expression such as "*/15 * * * *"`,
		},
		{
			name:       "unknown descriptor",
			expression: "@fortnightly",
			reason:     `unknown descriptor "@fortnightly"; supported descriptors are @hourly, @daily, @midnight, @weekly, @monthly, @yearly and @annually`,
		},
		{
			name:       "descriptor with arguments",
			expression: "@daily extra",
			reason:     `descriptor "@daily" does not take arguments`,
		},
		{
			// A schedule the server would happily store and never run.
			name:       "impossible date",
			expression: "0 0 30 2 *",
			reason:     "schedule never occurs: no calendar date matches the day-of-month, month and day-of-week fields",
		},
		{
			name:       "impossible date in a short month",
			expression: "0 0 31 4 *",
			reason:     "schedule never occurs: no calendar date matches the day-of-month, month and day-of-week fields",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				CronCheckResult{Status: CronInvalid, Reason: test.reason},
				CheckCron(test.expression),
			)
		})
	}
}

func TestCheckCronUnsupportedDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		reason     string
	}{
		{
			name:       "last day of month",
			expression: "0 0 L * *",
			reason:     `day-of-month field "L": the "L" (last day) extension is not supported`,
		},
		{
			name:       "last weekday of month",
			expression: "0 0 LW * *",
			reason:     `day-of-month field "LW": the "L" (last day) extension is not supported`,
		},
		{
			name:       "nearest weekday",
			expression: "0 0 15W * *",
			reason:     `day-of-month field "15W": the "W" (nearest weekday) extension is not supported`,
		},
		{
			name:       "extension inside an otherwise valid list",
			expression: "0 0 1,15W * *",
			reason:     `day-of-month field "1,15W": the "W" (nearest weekday) extension is not supported`,
		},
		{
			name:       "nth weekday",
			expression: "0 0 * * 5#3",
			reason:     `day-of-week field "5#3": the "#" (nth weekday of the month) extension is not supported`,
		},
		{
			name:       "no specific value",
			expression: "0 0 ? * MON",
			reason:     `day-of-month field "?": the "?" (no specific value) extension is not supported`,
		},
		{
			name:       "seconds field",
			expression: "0 0 0 * * *",
			reason:     "six-field expressions (leading seconds field) are not supported; use the five-field minute hour day-of-month month day-of-week form",
		},
		{
			name:       "seconds and year fields",
			expression: "0 0 0 * * * 2030",
			reason:     "seven-field expressions (leading seconds and trailing year fields) are not supported; use the five-field minute hour day-of-month month day-of-week form",
		},
		{
			// Shaped like a year, so the dialect is identified even though no
			// accepted year range has been established for the product.
			name:       "year outside any range we can vouch for",
			expression: "0 0 0 * * * 1969",
			reason:     "seven-field expressions (leading seconds and trailing year fields) are not supported; use the five-field minute hour day-of-month month day-of-week form",
		},
		{
			name:       "timezone directive",
			expression: "CRON_TZ=Asia/Kolkata 0 0 * * *",
			reason:     `timezone directives such as "CRON_TZ=Asia/Kolkata" are not supported; expressions are evaluated in UTC`,
		},
		{
			name:       "short timezone directive",
			expression: "TZ=UTC 0 0 * * *",
			reason:     `timezone directives such as "TZ=UTC" are not supported; expressions are evaluated in UTC`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				CronCheckResult{Status: CronUnsupportedDialect, Reason: test.reason},
				CheckCron(test.expression),
			)
		})
	}
}

type enumeration struct {
	violationGap int
	occurrence   time.Time
	next         time.Time
}

// enumerate walks real occurrences, which is exactly what the analysis refuses
// to do on the validate path. It is the independent check that reasoning over
// the field masks describes the schedule it claims to.
func enumerate(schedule *cronSchedule, origin time.Time, days int) enumeration {
	var (
		found    enumeration
		previous time.Time
		day      = origin.UTC().Truncate(24 * time.Hour)
	)

	for range days {
		if schedule.matchesDay(day) {
			for _, hour := range schedule.hours {
				for _, minute := range schedule.minutes {
					fire := at(day, hour, minute)
					if !previous.IsZero() && found.violationGap == 0 {
						if gap := int(fire.Sub(previous).Minutes()); gap < minGapMinutes {
							found.violationGap, found.occurrence, found.next = gap, previous, fire
						}
					}
					previous = fire
				}
			}
		}
		day = nextUTCDay(day)
	}

	return found
}

func TestCheckCronMatchesEnumeratedOccurrences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expression string
		days       int
	}{
		{expression: "* * * * *", days: 3},
		{expression: "*/5 * * * *", days: 3},
		{expression: "*/3 * * * *", days: 3},
		{expression: "0,10,20,30,40,50,53 * * * *", days: 3},
		{expression: "0,58 0,1 * * *", days: 3},
		{expression: "0,57 0,23 * * *", days: 3},
		{expression: "0,55 0,23 * * *", days: 400},
		{expression: "15/20 * * * *", days: 5},
		{expression: "0-30/10 * * * *", days: 5},
		{expression: "0 6 * * MON-FRI", days: 400},
		{expression: "0 0 13 * FRI", days: 400},
		{expression: "0 0 1 * *", days: 400},
		{expression: "0,3 0 29 2 *", days: 1200},
		{expression: "0,30 0 29 2 *", days: 1200},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()

			schedule, failure := parseCron(test.expression)
			require.Nil(t, failure)

			found := enumerate(schedule, cronOrigin, test.days)
			if found.violationGap > 0 {
				assert.Equal(t, tooFrequent(found.violationGap, found.occurrence, found.next), CheckCron(test.expression))
				return
			}

			assert.Equal(t, CronCheckResult{Status: CronValid}, CheckCron(test.expression))
		})
	}
}

// Day-of-week named WED contains the W that marks the "nearest weekday"
// extension, so the two must not be confused.
func TestCheckCronDayNameIsNotAnExtension(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CronCheckResult{Status: CronValid}, CheckCron("0 0 * * WED"))
	assert.Equal(t, CronCheckResult{Status: CronValid}, CheckCron("0 0 * * WED-FRI"))
}

// The analysis reads no clock and keeps no state, and the day scan has no
// cutoff: repeated evaluations must agree exactly, and no expression may fall
// back to CronInconclusive.
func TestCheckCronIsRepeatableAndConclusive(t *testing.T) {
	t.Parallel()

	var (
		first    []CronCheckResult
		second   []CronCheckResult
		statuses []CronStatus
	)
	for _, expression := range []string{
		"* * * * *", "*/5 * * * *", "0,3 0 29 2 *", "0 0 30 2 *",
		"@daily", "@every 1h", "0 0 L * *", "a b c d e f",
	} {
		result := CheckCron(expression)
		first = append(first, result)
		second = append(second, CheckCron(expression))
		statuses = append(statuses, result.Status)
	}

	assert.Equal(t, first, second)
	assert.NotContains(t, statuses, CronInconclusive)
}

// The leap-day violation is a property of the minute field, so moving the
// origin lands on a different leap day with the same verdict: the origin only
// moves the timestamps.
func TestCheckCronOriginOnlyMovesTimestamps(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CronCheckResult{
		Status:         CronTooFrequent,
		Reason:         "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
		Occurrence:     utc(2032, time.February, 29, 0, 0),
		NextOccurrence: utc(2032, time.February, 29, 0, 3),
	}, checkCronFrom("0,3 0 29 2 *", time.Date(2029, time.January, 1, 0, 0, 0, 0, time.UTC)))
}

// An impossible date is the worst case: it is only provable after the full
// 146097-day scan, so this is the number the package doc quotes.
func BenchmarkCheckCronWorstCase(b *testing.B) {
	for b.Loop() {
		CheckCron("0 0 30 2 *")
	}
}
