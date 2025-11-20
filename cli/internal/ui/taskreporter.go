package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBlue))

type taskState struct {
	id      string
	message string
}

// Custom messages
type taskStartMsg struct {
	id      string
	message string
}

type taskCompleteMsg struct {
	id      string
	message string
	err     error
}

type tasksDoneMsg struct{}

// Main model
type model struct {
	tasksMsgChan chan tea.Msg
	tasks        map[string]*taskState
	total        int
	completed    int
	sp           spinner.Model
	pr           progress.Model
}

func initialModel(totalTasks int) model {
	return model{
		tasksMsgChan: make(chan tea.Msg),
		tasks:        make(map[string]*taskState),
		total:        totalTasks,
		sp:           spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(spinnerStyle)),
		pr: progress.New(
			progress.WithSolidFill(ColorWhite),
			progress.WithFillCharacters('#', '_'),
			progress.WithoutPercentage(),
		),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.sp.Tick, listenForTaskMessages(m.tasksMsgChan))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case taskStartMsg:
		m.tasks[msg.id] = &taskState{id: msg.id, message: msg.message}
		return m, listenForTaskMessages(m.tasksMsgChan)

	case taskCompleteMsg:
		delete(m.tasks, msg.id)
		m.completed++

		var completionMsg string
		if msg.err != nil {
			completionMsg = Failure(msg.message)
		} else {
			completionMsg = Success(msg.message)
		}

		return m, tea.Batch(
			tea.Println(completionMsg),
			listenForTaskMessages(m.tasksMsgChan),
		)

	case tasksDoneMsg:
		return m, tea.Quit

	case spinner.TickMsg:
		msp, cmd := m.sp.Update(msg)
		m.sp = msp
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var output strings.Builder

	for _, task := range m.tasks {
		output.WriteString(fmt.Sprintf("%s %s\n", m.sp.View(), task.message))
	}

	if len(m.tasks) > 0 {
		output.WriteString(fmt.Sprintf("[%s] %d/%d\n", m.pr.ViewAs(float64(m.completed)/float64(m.total)), m.completed, m.total))
	}

	return output.String()
}

// listenForTaskMessages creates a command that reads from the channel
func listenForTaskMessages(msgChan chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-msgChan
	}
}

type TaskReporter struct {
	m model
}

func (t *TaskReporter) Run() error {
	p := tea.NewProgram(t.m)

	_, err := p.Run()
	return err
}

func NewTaskReporter(totalTasks int) *TaskReporter {
	return &TaskReporter{
		m: initialModel(totalTasks),
	}
}

func (t *TaskReporter) Start(id string, message string) {
	t.m.tasksMsgChan <- taskStartMsg{id: id, message: message}
}

func (t *TaskReporter) Complete(id string, message string, err error) {
	t.m.tasksMsgChan <- taskCompleteMsg{id: id, message: message, err: err}
}

func (t *TaskReporter) Done() {
	t.m.tasksMsgChan <- tasksDoneMsg{}
}
