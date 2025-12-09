package ui

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/assert"
)

func TestTaskReporter_Model(t *testing.T) {
	trm := initialModel(3)
	tm := teatest.NewTestModel(t, trm)

	ch := trm.tasksMsgChan
	bts, _ := io.ReadAll(tm.Output())
	assert.Empty(t, bts, "expected no output initially")

	ch <- taskStartMsg{id: "task1", message: "Processing task 1"}
	ch <- taskStartMsg{id: "task2", message: "Processing task 2"}

	expectOuput(t, tm, []string{
		expectedProcessingMsg("1"),
		expectedProcessingMsg("2"),
		"0/3",
	}, nil)

	ch <- taskCompleteMsg{id: "task1", message: "Completed task 1", err: nil}

	expectOuput(t, tm, []string{
		expectedCompletedMsg("1"),
		expectedProcessingMsg("2"),
		"1/3",
	}, []string{
		expectedProcessingMsg("1"),
	})

	ch <- taskCompleteMsg{id: "task2", message: "Failed task 2", err: fmt.Errorf("some error")}

	expectOuput(t, tm, []string{
		expectedFailedMsg("2"),
		"2/3",
	}, []string{
		expectedProcessingMsg("1"),
		expectedProcessingMsg("2"),
	})

	ch <- taskStartMsg{id: "task3", message: "Processing task 3"}
	ch <- taskCompleteMsg{id: "task3", message: "Completed task 3", err: nil}

	expectOuput(t, tm, []string{
		expectedCompletedMsg("3"),
	}, []string{
		expectedProcessingMsg("3"),
		"3/3",
	})

	ch <- tasksDoneMsg{}
	tm.WaitFinished(t)
	finalReader := tm.FinalOutput(t)
	finalBts, _ := io.ReadAll(finalReader)

	assert.NotContains(t, string(finalBts), "Processing task", "no processing tasks should remain in final output")
	assert.NotContains(t, string(finalBts), "/3", "no progress bar should remain in final output")
}

func expectOuput(t *testing.T, tm *teatest.TestModel, shouldContain []string, shouldNotContain []string) {
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			checks := true
			for _, str := range shouldContain {
				if !bytes.Contains(bts, []byte(str)) {
					checks = false
				}
			}

			for _, str := range shouldNotContain {
				if bytes.Contains(bts, []byte(str)) {
					checks = false
				}
			}

			return checks
		},
		teatest.WithCheckInterval(10*time.Millisecond),
	)
}

func expectedProcessingMsg(ID string) string {
	// spinner output is the same for both tasks
	return fmt.Sprintf("⠋ Processing task %s", ID)
}

func expectedCompletedMsg(ID string) string {
	return fmt.Sprintf("✔ Completed task %s", ID)
}

func expectedFailedMsg(ID string) string {
	return fmt.Sprintf("x Failed task %s", ID)
}

func TestTaskReporter(t *testing.T) {
	tr := NewTaskReporter(1)

	go func() {
		tr.Start("task1", "Processing task 1")
		tr.Complete("task1", "Completed task 1", nil)
		tr.Done()
		close(tr.m.tasksMsgChan)
	}()

	receivedMessages := make([]tea.Msg, 0)

	for msg := range tr.m.tasksMsgChan {
		receivedMessages = append(receivedMessages, msg)
	}

	assert.Len(t, receivedMessages, 3, "expected 3 messages to be sent")
	assert.Equal(t, taskStartMsg{"task1", "Processing task 1"}, receivedMessages[0])
	assert.Equal(t, taskCompleteMsg{"task1", "Completed task 1", nil}, receivedMessages[1])
	assert.Equal(t, tasksDoneMsg{}, receivedMessages[2])
}
