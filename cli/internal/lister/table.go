package lister

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.DoubleBorder()).
	BorderForeground(lipgloss.Color("240"))

type tableModel struct {
	table      table.Model
	windowSize tea.WindowSizeMsg
}

func (m tableModel) Init() tea.Cmd { return nil }

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.table.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m tableModel) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

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

func printResourcesAsBubbleTeaTable(rs []resources.ResourceData) error {
	if len(rs) == 0 {
		return nil
	}

	headers := getHeaders(rs)
	columns := make([]table.Column, len(headers))
	for i, h := range headers {
		columns[i] = table.Column{Title: h, Width: len(h)}
	}

	rows := make([]table.Row, len(rs))
	for i, r := range rs {
		row, err := getTableRow(r, headers)
		if err != nil {
			return fmt.Errorf("failed to create table row: %w", err)
		}
		rows[i] = row
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rs)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := tableModel{table: t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("failed to run bubbletea program: %w", err)
	}

	return nil
}
