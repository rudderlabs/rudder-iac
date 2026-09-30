package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// evidencePageLimit is the page size dev exec reads the matched events with.
const evidencePageLimit = 1000

type evidenceTruncated struct {
	By      string `json:"by"`
	Kept    int    `json:"kept"`
	Dropped int    `json:"dropped"`
	Next    string `json:"next"`
}

// expectedNames lists each expected event name once, in --expect order.
func expectedNames(expected []devlisten.Expected) []string {
	var names []string
	for _, e := range expected {
		if !slices.Contains(names, e.Event) {
			names = append(names, e.Event)
		}
	}
	return names
}

// collectEvents reads every page of q with no byte cap. The listener is
// about to stop, so there is no later call to continue from.
func collectEvents(ctx context.Context, c *devlisten.Client, q devlisten.Query) ([]json.RawMessage, error) {
	q.Limit, q.MaxBytes = evidencePageLimit, devlisten.MaxBytesOff
	var items []json.RawMessage
	for {
		page, err := c.Events(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, ev := range page.Events {
			items = append(items, ev.Raw)
		}
		if !page.HasMore || page.Cursor <= q.Since {
			return items, nil
		}
		q.Since = page.Cursor
	}
}

// boundItems drops whole items, last first, until the items fit in
// maxBytes; 0 turns the cap off.
func boundItems(items []json.RawMessage, maxBytes int) ([]json.RawMessage, int) {
	size := 0
	for _, item := range items {
		size += len(item) + 1
	}
	n := len(items)
	for maxBytes > 0 && n > 0 && size > maxBytes {
		n--
		size -= len(items[n]) + 1
	}
	return items[:n], len(items) - n
}

// withEvents adds items and, when some were dropped, truncated to the
// events object of a summary. Re-encoding sorts every object key.
func withEvents(summary []byte, items []json.RawMessage, t *evidenceTruncated) ([]byte, error) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(summary, &env); err != nil {
		return nil, fmt.Errorf("decoding summary: %w", err)
	}
	var events map[string]json.RawMessage
	if err := json.Unmarshal(env["events"], &events); err != nil {
		return nil, fmt.Errorf("decoding summary events: %w", err)
	}
	if items == nil {
		items = []json.RawMessage{}
	}
	events["items"] = encodeJSON(items)
	if t != nil {
		events["truncated"] = encodeJSON(t)
	}
	env["events"] = encodeJSON(events)
	return encodeJSON(env), nil
}

// encodeJSON marshals without HTML escaping, as the server does.
func encodeJSON(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
