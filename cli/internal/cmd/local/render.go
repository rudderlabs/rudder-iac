package local

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

// maxCell caps a table cell, so one large property keeps the table readable.
const maxCell = 60

// summaryText holds the summary keys the text form shows. The JSON form is
// the contract; this is for people.
type summaryText struct {
	Since          uint64 `json:"since"`
	Cursor         uint64 `json:"cursor"`
	TimedOut       bool   `json:"timedOut"`
	WaitedMs       int64  `json:"waitedMs"`
	EvictedThrough uint64 `json:"evictedThrough"`
	Summary        struct {
		Requests struct {
			Total  int `json:"total"`
			Failed int `json:"failed"`
		} `json:"requests"`
		Events struct {
			Total int `json:"total"`
		} `json:"events"`
		ByEvent  map[string]int `json:"byEvent"`
		Rejected struct {
			Events  int            `json:"events"`
			ByEvent map[string]int `json:"byEvent"`
		} `json:"rejected"`
		Control struct {
			Total int `json:"total"`
		} `json:"control"`
		Diagnosis []struct {
			Code    string `json:"code"`
			Count   int    `json:"count"`
			Message string `json:"message"`
			Next    string `json:"next"`
		} `json:"diagnosis"`
	} `json:"summary"`
	Next string `json:"next"`
}

func (s *summaryText) render(out io.Writer) error {
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "Window after cursor %d; cursor now %d.", s.Since, s.Cursor)
	if s.TimedOut {
		fmt.Fprintf(w, " The wait ran out after %.1fs.", float64(s.WaitedMs)/1000)
	}
	fmt.Fprint(w, "\n\n")

	fmt.Fprintf(w, "ACCEPTED EVENTS  %d\n", s.Summary.Events.Total)
	counts(w, s.Summary.ByEvent)
	fmt.Fprintf(w, "REJECTED EVENTS  %d\n", s.Summary.Rejected.Events)
	counts(w, s.Summary.Rejected.ByEvent)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "REQUESTS\t%d (%d failed)\n", s.Summary.Requests.Total, s.Summary.Requests.Failed)
	fmt.Fprintf(tw, "CONTROL\t%d\n", s.Summary.Control.Total)
	_ = tw.Flush()

	if len(s.Summary.Diagnosis) > 0 {
		fmt.Fprintln(w)
	}
	for _, d := range s.Summary.Diagnosis {
		fmt.Fprintf(w, "%s (%d): %s\n  Next: %s\n", d.Code, d.Count, d.Message, d.Next)
	}
	fmt.Fprintf(w, "\nNext: %s\n", s.Next)
	return w.Flush()
}

// counts lists names by count, most first, so the events that arrived lead
// and the zero entries a caller asked for follow.
func counts(w io.Writer, byName map[string]int) {
	names := slices.SortedFunc(maps.Keys(byName), func(a, b string) int {
		return cmp.Or(cmp.Compare(byName[b], byName[a]), cmp.Compare(a, b))
	})
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, name := range names {
		fmt.Fprintf(tw, "  %s\t%d\n", cell(name), byName[name])
	}
	_ = tw.Flush()
}

type streamTable struct {
	since  string
	base   string
	more   string
	view   string
	fields []string
}

func renderStream(out io.Writer, body []byte, t streamTable) error {
	var events []gjson.Result
	for line := range bytes.Lines(body) {
		if len(bytes.TrimSpace(line)) > 0 {
			events = append(events, gjson.ParseBytes(line))
		}
	}
	w := bufio.NewWriter(out)
	window := "after cursor " + t.since
	if _, err := strconv.ParseUint(t.since, 10, 64); err != nil {
		window = "since " + t.since
	}
	if len(events) == 0 {
		fmt.Fprintf(w, "0 events %s; run rudder-cli local event-stream events summary --url %s --since %s for the diagnosis\n",
			window, shellWord(t.base), shellWord(t.since))
		return w.Flush()
	}
	fmt.Fprintf(w, "%d events %s", len(events), window)
	if t.more != "" {
		fmt.Fprintf(w, "; more with --since %s", t.more)
	}
	fmt.Fprintln(w)

	var table bytes.Buffer
	tw := tabwriter.NewWriter(&table, 0, 0, 2, ' ', 0)
	header := []string{"TIME", "TYPE", "EVENT", "USER"}
	if t.view == "compact" {
		header = append(header, "PROPERTIES")
	}
	if len(t.fields) > 0 {
		header = header[:0]
		for _, f := range t.fields {
			header = append(header, strings.ToUpper(f))
		}
	}
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, ev := range events {
		fmt.Fprintln(tw, strings.Join(t.row(ev), "\t"))
	}
	_ = tw.Flush()
	// tabwriter pads an empty last cell; a line with trailing blanks
	// breaks a copy and paste.
	for line := range bytes.Lines(table.Bytes()) {
		w.Write(bytes.TrimRight(line, " \n"))
		w.WriteByte('\n')
	}
	return w.Flush()
}

// row shows the event's own keys only: the stream carries no capture data,
// so the time is the SDK's.
func (t streamTable) row(ev gjson.Result) []string {
	if len(t.fields) > 0 {
		row := make([]string, len(t.fields))
		for i, path := range t.fields {
			row[i] = value(lookup(ev, path))
		}
		return row
	}
	row := []string{
		first(ev, "originalTimestamp", "sentAt", "timestamp"),
		first(ev, "type"),
		first(ev, "event", "name"),
		first(ev, "userId", "anonymousId"),
	}
	if t.view == "compact" {
		row = append(row, value(ev.Get("properties")))
	}
	return row
}

// lookup walks a dotted path key by key, so gjson path syntax in a key
// selects nothing by accident.
func lookup(ev gjson.Result, path string) gjson.Result {
	for _, key := range strings.Split(path, ".") {
		ev = ev.Get(gjson.Escape(key))
	}
	return ev
}

func first(ev gjson.Result, keys ...string) string {
	for _, key := range keys {
		if v := ev.Get(gjson.Escape(key)); v.Exists() {
			return value(v)
		}
	}
	return ""
}

// value shows a string as text and anything else as its JSON.
func value(v gjson.Result) string {
	if !v.Exists() {
		return ""
	}
	if v.Type == gjson.String {
		return cell(v.Str)
	}
	return cell(v.Raw)
}

// cell escapes control characters, terminal escapes included, because a
// captured value comes from any page that can post to the listener, and caps
// the width.
func cell(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n == maxCell-1 && utf8.RuneCountInString(s) > maxCell {
			b.WriteString("…")
			break
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			fmt.Fprintf(&b, `\u%04x`, r)
		} else {
			b.WriteRune(r)
		}
		n++
	}
	return b.String()
}
