package lister

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func printResourcesAsTable(rs []resources.ResourceData) error {
	if len(rs) == 0 {
		return nil
	}

	headers := getHeaders(rs)
	data := make([][]string, len(rs))
	for i, r := range rs {
		row, err := getTableRow(r, headers)
		if err != nil {
			return fmt.Errorf("failed to create table row: %w", err)
		}
		data[i] = row
	}

	columnWidths := make([]int, len(headers))
	for i, h := range headers {
		columnWidths[i] = len(h)
	}
	for _, row := range data {
		for i, cell := range row {
			if len(cell) > columnWidths[i] {
				columnWidths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Fprintf(os.Stdout, "%-*s  ", columnWidths[i], h)
	}
	fmt.Fprintln(os.Stdout)

	// Print separator
	for _, w := range columnWidths {
		fmt.Fprint(os.Stdout, strings.Repeat("-", w)+"  ")
	}
	fmt.Fprintln(os.Stdout)

	// Print data
	for _, row := range data {
		for i, cell := range row {
			fmt.Fprintf(os.Stdout, "%-*s  ", columnWidths[i], cell)
		}
		fmt.Fprintln(os.Stdout)
	}

	return nil
}

func getHeaders(rs []resources.ResourceData) []string {
	headerMap := make(map[string]struct{})
	for _, r := range rs {
		for k := range r {
			headerMap[k] = struct{}{}
		}
	}

	headers := make([]string, 0, len(headerMap))
	for k := range headerMap {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	// Move "id" and "name" to the beginning if they exist
	for _, key := range []string{"name", "id"} {
		for i, h := range headers {
			if h == key {
				headers = append(headers[:i], headers[i+1:]...)
				headers = append([]string{key}, headers...)
				break
			}
		}
	}

	// Move "createdAt", "updatedAt", and "definition" to the end if they exist
	for _, key := range []string{"createdAt", "updatedAt"} {
		for i, h := range headers {
			if h == key {
				headers = append(headers[:i], headers[i+1:]...)
				headers = append(headers, key)
				break
			}
		}
	}

	return headers
}

func getTableRow(r resources.ResourceData, headers []string) ([]string, error) {
	row := make([]string, len(headers))
	for i, h := range headers {
		val, ok := r[h]
		if !ok {
			row[i] = ""
			continue
		}

		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Map || v.Kind() == reflect.Slice {
			b, err := json.Marshal(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal field %s: %w", h, err)
			}
			row[i] = string(b)
		} else {
			row[i] = fmt.Sprintf("%v", val)
		}
	}
	return row, nil
}
