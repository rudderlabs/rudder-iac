package sources

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

// maxCell caps a table cell, so one large property keeps the table readable.
const maxCell = 60

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
		fmt.Fprintf(w, "0 events %s; refused requests: curl -fsS %s\n",
			window, shellWord(t.base+"/_dev/v1/requests?since="+url.QueryEscape(t.since)+"&failed=true&view=compact"))
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
		row = append(row, value(lastValue(ev, "properties")))
	}
	return row
}

// lookup walks a dotted path key by key, so gjson path syntax in a key
// selects nothing by accident.
func lookup(ev gjson.Result, path string) gjson.Result {
	if ev.IsArray() {
		// The API's field selection keeps every element, so the table does
		// too: an object takes the rest of the path, anything else stays as
		// sent, and a missing key shows null to keep the positions.
		var values []string
		for _, item := range ev.Array() {
			if !item.IsObject() {
				values = append(values, item.Raw)
				continue
			}
			if v := lookup(item, path); v.Exists() {
				values = append(values, v.Raw)
				continue
			}
			values = append(values, "null")
		}
		return gjson.Parse("[" + strings.Join(values, ",") + "]")
	}
	key, rest, more := strings.Cut(path, ".")
	ev = lastValue(ev, key)
	if more {
		return lookup(ev, rest)
	}
	return ev
}

// lastValue matches the last-key-wins lookup used by filters and field selection.
func lastValue(ev gjson.Result, key string) gjson.Result {
	var found gjson.Result
	if ev.IsObject() {
		ev.ForEach(func(k, v gjson.Result) bool {
			if k.Str == key {
				found = v
			}
			return true
		})
	}
	return found
}

func first(ev gjson.Result, keys ...string) string {
	for _, key := range keys {
		if v := lastValue(ev, key); v.Exists() {
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
