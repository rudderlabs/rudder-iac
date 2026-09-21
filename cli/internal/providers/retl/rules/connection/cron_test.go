package connection

import (
	"strings"
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
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 1-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 1, 0, 0),
				NextViolatingSync: utc(2026, time.January, 1, 0, 1),
			},
		},
		{
			expression: "*/3 * * * *",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 1, 0, 0),
				NextViolatingSync: utc(2026, time.January, 1, 0, 3),
			},
		},
		{
			// An irregular minute list only breaks the floor at the tail of the
			// list, which an "every N minutes" reading of the field would miss.
			expression: "0,10,20,30,40,50,53 * * * *",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 1, 0, 50),
				NextViolatingSync: utc(2026, time.January, 1, 0, 53),
			},
		},
		{
			// The violation is the join between two selected hours, not a gap
			// inside either of them.
			expression: "0,58 0,1 * * *",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 2-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 1, 0, 58),
				NextViolatingSync: utc(2026, time.January, 1, 1, 0),
			},
		},
		{
			// The violation only shows up across midnight.
			expression: "0,57 0,23 * * *",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 1, 23, 57),
				NextViolatingSync: utc(2026, time.January, 2, 0, 0),
			},
		},
		{
			// Exactly five minutes across midnight is accepted.
			expression: "0,55 0,23 * * *",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			// A literal "*" anywhere in the field sets the flag, which is
			// robfig's rule, so both spellings are wildcards and select the
			// same schedule here as they do there. The day fields are ANDed,
			// only Mondays match - never two days running, which is what keeps
			// the midnight gap out of reach.
			expression: "0,57 0,23 3,* * MON",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			expression: "0,57 0,23 *,3 * MON",
			expected:   CronCheckResult{Status: CronValid},
		},
		{
			// A stepped wildcard clears the flag, as it does in robfig, so the
			// day fields are ORed: every odd day matches and so does every
			// Monday. The first odd day followed by a Monday puts three minutes
			// across midnight, and robfig fires exactly this pair.
			expression: "0,57 0,23 */2 * MON",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2026, time.January, 11, 23, 57),
				NextViolatingSync: utc(2026, time.January, 12, 0, 0),
			},
		},
		{
			// Leap day: the reference origin is 2026, so the first occurrence is
			// two years out and the violation still has to be reported.
			expression: "0,3 0 29 2 *",
			expected: CronCheckResult{
				Status:            CronTooFrequent,
				Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
				ViolatingSync:     utc(2028, time.February, 29, 0, 0),
				NextViolatingSync: utc(2028, time.February, 29, 0, 3),
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
			// rETL has no reboot to fire on. The descriptor table is what makes
			// that an error rather than a warning, so @reboot needs no case of
			// its own to be rejected.
			name:       "reboot descriptor",
			expression: "@reboot",
			reason:     `unknown descriptor "@reboot"; supported descriptors are @hourly, @daily, @midnight, @weekly, @monthly, @yearly and @annually`,
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
			// A day number outside the field's range is malformed in every
			// dialect, so the extension shape must not excuse it. These are the
			// cases where an error was being downgraded to a warning, which
			// under DEX-829 would apply a schedule that can never run.
			name:       "out of range day-of-week in # extension",
			expression: "0 0 * * 9#3",
			reason:     `day-of-week field "9#3": "9#3" is not a number or day-of-week name`,
		},
		{
			name:       "out of range day-of-month in L extension",
			expression: "0 0 32L * *",
			reason:     `day-of-month field "32L": "32L" is not a number`,
		},
		{
			name:       "out of range day-of-month in W extension",
			expression: "0 0 0W * *",
			reason:     `day-of-month field "0W": "0W" is not a number`,
		},
		{
			name:       "out of range offset in L- extension",
			expression: "0 0 L-99 * *",
			reason:     `day-of-month field "L-99": "L" is not a number`,
		},
		{
			name:       "out of range day-of-week in L extension",
			expression: "0 0 * * 8L",
			reason:     `day-of-week field "8L": "8L" is not a number or day-of-week name`,
		},
		{
			// Quartz spells these with day names too. Only the numeric forms
			// are recognised, and erring towards an error is the safe
			// direction: a warning would let the schedule through unchecked.
			name:       "day name spelling of the # extension",
			expression: "0 0 * * FRI#2",
			reason:     `day-of-week field "FRI#2": "FRI#2" is not a number or day-of-week name`,
		},
		{
			name:       "day name spelling of the L extension",
			expression: "0 0 * * FRIL",
			reason:     `day-of-week field "FRIL": "FRIL" is not a number or day-of-week name`,
		},
		{
			// Quartz writes "?" only in the two day fields. A parser that read
			// it as "*" anywhere would widen this to every minute of every day
			// instead of rejecting it, so the boundary is pinned here.
			name:       "no specific value outside the day fields",
			expression: "? 1 7 1 5",
			reason:     `minute field "?": "?" is not a number`,
		},
		{
			// A directive is reported only once the expression it prefixes has
			// been checked, or it would shield anything written after it behind
			// a warning that lets the apply through.
			name:       "timezone directive in front of a malformed field",
			expression: "TZ=UTC 0 0 * * 9#3",
			reason:     `day-of-week field "9#3": "9#3" is not a number or day-of-week name`,
		},
		{
			name:       "timezone directive with no expression",
			expression: "TZ=UTC",
			reason:     "timezone directive is not followed by a cron expression",
		},
		{
			// The year is checked even though the day field already produced a
			// warning, so an extension cannot carry a malformed year past it.
			name:       "extension does not excuse a malformed year",
			expression: "0 0 0 L * ? garbage",
			reason:     `year field "garbage": "garbage" is not a number`,
		},
		{
			// Quartz writes "#" in day-of-week only. Accepted in day-of-month
			// it would check a weekday number against the day numbers 1-31 and
			// pass a weekday that does not exist.
			name:       "nth weekday in the day-of-month field",
			expression: "0 0 9#3 * *",
			reason:     `day-of-month field "9#3": "9#3" is not a number`,
		},
		{
			name:       "nearest weekday in the day-of-week field",
			expression: "0 0 * * 5W",
			reason:     `day-of-week field "5W": "5W" is not a number or day-of-week name`,
		},
		{
			name:       "last given weekday in the day-of-month field",
			expression: "0 0 31L * *",
			reason:     `day-of-month field "31L": "31L" is not a number`,
		},
		{
			// An L- offset runs 0-30, not over the day numbers, so 31 could
			// never select a day no matter how long the month.
			name:       "out of range offset in L- extension",
			expression: "0 0 L-31 * *",
			reason:     `day-of-month field "L-31": "L" is not a number`,
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
			reason:     `day-of-month field "LW": the "LW" (last weekday of the month) extension is not supported`,
		},
		{
			// Quartz writes this one in day-of-week: the last Friday of the
			// month. It is a dialect we decline to analyse, not malformed.
			name:       "last given weekday of month",
			expression: "0 0 * * 5L",
			reason:     `day-of-week field "5L": the "L" (last given weekday of the month) extension is not supported`,
		},
		{
			// The lower bound of the offset range, which is 0 and not 1.
			name:       "zero offset from the last day",
			expression: "0 0 L-0 * *",
			reason:     `day-of-month field "L-0": the "L-n" (offset from the last day) extension is not supported`,
		},
		{
			name:       "extension inside a seven-field expression",
			expression: "0 0 0 L * ? 2030",
			reason:     `day-of-month field "L": the "L" (last day) extension is not supported`,
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
// to do on the validate path. It shares the parser and the day rule with the
// implementation, so it vouches for neither: what it cross-checks is that the
// three-join-point shortcut reports the same first violation a full walk finds.
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

// The leap-day violation is a property of the minute field, so moving the
// origin lands on a different leap day with the same verdict: the origin only
// moves the timestamps.
func TestCheckCronOriginOnlyMovesTimestamps(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CronCheckResult{
		Status:            CronTooFrequent,
		Reason:            "consecutive syncs have a 3-minute gap; the minimum supported interval is 5 minutes",
		ViolatingSync:     utc(2032, time.February, 29, 0, 0),
		NextViolatingSync: utc(2032, time.February, 29, 0, 3),
	}, checkCronFrom("0,3 0 29 2 *", time.Date(2029, time.January, 1, 0, 0, 0, 0, time.UTC)))
}

// Two shapes walk the whole 146097-day cycle: a date that never occurs, which
// cannot be disproved any sooner, and a valid schedule whose matching days are
// never calendar-adjacent. Neither is a pathological input, so both are pinned.
func BenchmarkCheckCron(b *testing.B) {
	for _, expression := range []string{"0 0 30 2 *", "0,57 0,23 29 2 *"} {
		b.Run(expression, func(b *testing.B) {
			for b.Loop() {
				CheckCron(expression)
			}
		})
	}
}

// A field of repeated extensions used to be formatted into a diagnostic once
// per matching element while only the first was ever kept, which is quadratic
// in the length of the field: 8000 elements allocated 252MB. Only the first
// match is built now, and this is what would show a regression.
func BenchmarkCheckCronRepeatedExtensions(b *testing.B) {
	expression := "0 0 " + strings.Repeat("L,", 4000) + "garbage * *"

	for b.Loop() {
		CheckCron(expression)
	}
}
