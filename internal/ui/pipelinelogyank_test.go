package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func yankLogsTestModel(t *testing.T) (Model, *mockPipelineProvider) {
	t.Helper()
	failedRun := testRun(domain.PipelineStatusFailed)
	provider := &mockPipelineProvider{
		run:  &failedRun,
		logs: testLogs(domain.PipelineStatusFailed),
		logLines: map[int][]string{
			testBuildLogID:  {"compiling", "compiled"},
			testDeployLogID: logLinesWithError(3, 2),
		},
	}
	return openedLogsModel(t, provider, failedRun), provider
}

func TestYankStepLog_CopiesTheShownStep(t *testing.T) {
	copied := recordClipboard(t)
	m, provider := yankLogsTestModel(t)

	m, _ = pressKey(t, m, keyYankPrefix)
	m, _ = pressKey(t, m, keyYankStepLog)

	if want := strings.Join(provider.logLines[testDeployLogID], "\n"); len(*copied) != 1 || (*copied)[0] != want {
		t.Errorf("expected the Deploy step's log copied, got %q", *copied)
	}
	if !strings.Contains(m.statusBar.Message(), "Copied 3 lines of Deploy") {
		t.Errorf("expected the copy reported, got %q", m.statusBar.Message())
	}
}

func TestYankRunLogs_ReadsAndCopiesEveryStepInRunOrder(t *testing.T) {
	copied := recordClipboard(t)
	m, provider := yankLogsTestModel(t)

	m, _ = pressKey(t, m, keyYankPrefix)
	m, cmd := pressKey(t, m, keyYankRunLogs)
	m, _ = m.handlePipelineRunLogsRead(findMsg[PipelineRunLogsReadMsg](t, cmd))

	logs := testLogs(domain.PipelineStatusFailed)
	want := fmt.Sprintf(pipelineLogYankHeaderFormat, logs[0].Path()) + "\n" + strings.Join(provider.logLines[testBuildLogID], "\n") +
		"\n\n" + fmt.Sprintf(pipelineLogYankHeaderFormat, logs[1].Path()) + "\n" + strings.Join(provider.logLines[testDeployLogID], "\n")
	if len(*copied) != 1 || (*copied)[0] != want {
		t.Errorf("expected both steps copied in run order, got %q", *copied)
	}
	if !strings.Contains(m.statusBar.Message(), "Copied the logs of 2 steps") {
		t.Errorf("expected the copy reported, got %q", m.statusBar.Message())
	}
}

func TestYankLogs_AreHintedInTheTopBar(t *testing.T) {
	m, _ := yankLogsTestModel(t)
	hint := fmt.Sprintf("<%s%s/%s%s> Copy step log / run logs", keyYankPrefix, keyYankStepLog, keyYankPrefix, keyYankRunLogs)

	if !slices.Contains(m.commandRegistry.GetContextualShortcuts(ViewPipelineLog, false), hint) {
		t.Errorf("expected %q among the log view's shortcuts", hint)
	}
	if slices.Contains(m.commandRegistry.GetContextualShortcuts(ViewPRList, false), hint) {
		t.Error("expected the log yanks only in the log view")
	}
}
