package ui

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

func createTestModel() Model {
	return Model{
		state:             ViewPRInspect,
		topBar:            components.NewTopBar(),
		statusBar:         components.NewStatusBar(),
		commandBar:        components.NewCommandBar(),
		patsView:          views.NewPATsView(),
		prListView:        views.NewPRListView(),
		prInspect:         views.NewPRInspectView(),
		reviewView:        views.NewReviewView(),
		inlineCommentView: views.NewInlineCommentView(),
		commentDetailView: views.NewCommentDetailView(),
		logsView:          views.NewLogsView(),
		commandRegistry:   NewCommandRegistry(),
	}
}

func TestHandleViewDiffKey_SwitchesToDiffMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	newModel, _ := handleViewDiffKey(m)

	if newModel.prInspect.GetMode() != views.PRInspectModeDiff {
		t.Errorf("expected mode to be PRInspectModeDiff, got %v", newModel.prInspect.GetMode())
	}
}

func TestHandleViewDiffKey_OnlyWorksInPRInspectView(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList
	m.prInspect.SwitchToDescription()

	newModel, _ := handleViewDiffKey(m)

	if newModel.prInspect.GetMode() != views.PRInspectModeDescription {
		t.Error("expected mode to remain in description when not in ViewPRInspect")
	}
}

func TestHandleQuitKey_InDiffMode_SwitchesToDescription(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDiff()

	newModel, _ := handleQuitKey(m)

	if newModel.prInspect.GetMode() != views.PRInspectModeDescription {
		t.Errorf("expected mode to switch to description, got %v", newModel.prInspect.GetMode())
	}

	if newModel.state != ViewPRInspect {
		t.Errorf("expected to remain in ViewPRInspect, got %v", newModel.state)
	}
}

func TestHandleQuitKey_InDescriptionMode_NavigatesBack(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	newModel, _ := handleQuitKey(m)

	if newModel.state != ViewPRList {
		t.Errorf("expected to navigate to ViewPRList, got %v", newModel.state)
	}
}

func TestHandleQuitKey_InPATsView_Quits(t *testing.T) {
	m := createTestModel()
	m.state = ViewPATs

	_, cmd := handleQuitKey(m)

	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestHandleNextFileKey_OnlyWorksInDiffMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{{Type: "add", Content: "+line1"}}},
				},
			},
			{
				NewPath: "file2.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{{Type: "add", Content: "+line2"}}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SwitchToDiff()

	viewBefore := m.prInspect.View()
	if !contains(viewBefore, "File 1/2") {
		t.Error("expected to start on file 1")
	}

	m.prInspect.SwitchToDescription()
	m2, _ := handleNextFileKey(m)

	m2.prInspect.SwitchToDiff()
	viewAfterInDescMode := m2.prInspect.View()
	if !contains(viewAfterInDescMode, "File 1/2") {
		t.Error("expected to remain on file 1 when calling NextFile in description mode")
	}

	m.prInspect.SwitchToDiff()
	m3, _ := handleNextFileKey(m)

	viewAfterNext := m3.prInspect.View()
	if !contains(viewAfterNext, "File 2/2") {
		t.Error("expected to be on file 2 after calling NextFile in diff mode")
	}
}

func TestHandlePrevFileKey_OnlyWorksInDiffMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{{Type: "add", Content: "+line1"}}},
				},
			},
			{
				NewPath: "file2.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{{Type: "add", Content: "+line2"}}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SwitchToDiff()
	m.prInspect.NextFile()

	viewOnFile2 := m.prInspect.View()
	if !contains(viewOnFile2, "File 2/2") {
		t.Error("expected to be on file 2")
	}

	m.prInspect.SwitchToDescription()
	m2, _ := handlePrevFileKey(m)

	m2.prInspect.SwitchToDiff()
	viewAfterDescMode := m2.prInspect.View()
	if !contains(viewAfterDescMode, "File 2/2") {
		t.Error("expected to remain on file 2 when calling PrevFile in description mode")
	}

	m.prInspect.SwitchToDiff()
	m3, _ := handlePrevFileKey(m)

	viewAfterPrev := m3.prInspect.View()
	if !contains(viewAfterPrev, "File 1/2") {
		t.Error("expected to be on file 1 after calling PrevFile in diff mode")
	}
}

func TestHandleViewCommentsKey_WorksInBothModes(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)
	m.commentDetailView.SetSize(80, 24)

	comment := domain.Comment{
		ID:       "comment1",
		Body:     "Test comment",
		FilePath: "file1.go",
		Line:     1,
		Author: domain.User{
			Username: "testuser",
		},
	}

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{{Type: "add", Content: "+line1"}}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SetComments([]domain.Comment{comment})

	// Test in description mode
	m.prInspect.SwitchToDescription()
	m2, _ := handleViewCommentsKey(m)

	if !m2.commentDetailView.IsActive() {
		t.Error("expected comment detail view to activate in description mode")
	}

	m.commentDetailView.Deactivate()

	// Test in diff mode
	m.prInspect.SwitchToDiff()
	m3, _ := handleViewCommentsKey(m)

	if !m3.commentDetailView.IsActive() {
		t.Error("expected comment detail view to activate in diff mode")
	}
}

func TestHandleApproveKey_WorksInPRInspect(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	m, _ = handleApproveKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to activate in description mode")
	}

	m.reviewView.Deactivate()
	m.prInspect.SwitchToDiff()
	m, _ = handleApproveKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to activate in diff mode")
	}
}

func TestHandleRequestChangesKey_WorksInPRInspect(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	m, _ = handleRequestChangesKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to activate in description mode")
	}

	m.reviewView.Deactivate()
	m.prInspect.SwitchToDiff()
	m, _ = handleRequestChangesKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to activate in diff mode")
	}
}

func TestHandleEnterKey_InPRInspect_OnlyWorksInDiffMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	newModel, _ := handleEnterKey(m)

	if newModel.reviewView.IsActive() {
		t.Error("expected review view to not activate in description mode")
	}

	m.prInspect.SwitchToDiff()
	newModel, _ = handleEnterKey(m)

	if !newModel.reviewView.IsActive() {
		t.Error("expected review view to activate in diff mode")
	}
}

func TestHandleEnterKey_InPRList_StartsInDescriptionMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList

	pr := &domain.PullRequest{
		ID:     "test-pr",
		Number: 42,
		Title:  "Test PR",
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
	}

	m.prListView.SetPRs([]domain.PullRequest{*pr})

	newModel, _ := handleEnterKey(m)

	if newModel.state != ViewPRInspect {
		t.Errorf("expected state to be ViewPRInspect, got %v", newModel.state)
	}

	if newModel.prInspect.GetMode() != views.PRInspectModeDescription {
		t.Errorf("expected mode to be PRInspectModeDescription, got %v", newModel.prInspect.GetMode())
	}
}

func TestNavigateBack_FromDiffMode_GoesToDescription(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDiff()

	newModel, _ := m.navigateBack()
	resultModel := newModel.(Model)

	if resultModel.prInspect.GetMode() != views.PRInspectModeDescription {
		t.Errorf("expected mode to switch to description, got %v", resultModel.prInspect.GetMode())
	}

	if resultModel.state != ViewPRInspect {
		t.Errorf("expected to remain in ViewPRInspect, got %v", resultModel.state)
	}
}

func TestNavigateBack_FromDescriptionMode_GoesToPRList(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SwitchToDescription()

	newModel, _ := m.navigateBack()
	resultModel := newModel.(Model)

	if resultModel.state != ViewPRList {
		t.Errorf("expected to navigate to ViewPRList, got %v", resultModel.state)
	}
}

func TestNavigateBack_FromPRList_GoesToPATs(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList

	newModel, _ := m.navigateBack()
	resultModel := newModel.(Model)

	if resultModel.state != ViewPATs {
		t.Errorf("expected to navigate to ViewPATs, got %v", resultModel.state)
	}
}

func TestKeyBindingsRegistered(t *testing.T) {
	registry := NewCommandRegistry()

	testCases := []struct {
		key         string
		description string
		viewState   ViewState
	}{
		{"d", "View diff", ViewPRInspect},
		{"left", "Previous file", ViewPRInspect},
		{"right", "Next file", ViewPRInspect},
	}

	for _, tc := range testCases {
		found := false
		for _, binding := range registry.keyBindings {
			for _, key := range binding.Keys {
				if key == tc.key && binding.Description == tc.description {
					availableIn := false
					for _, state := range binding.AvailableIn {
						if state == tc.viewState {
							availableIn = true
							break
						}
					}
					if availableIn {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("expected key binding for '%s' with description '%s' in view %v to be registered", tc.key, tc.description, tc.viewState)
		}
	}
}

func TestDKeyBinding_AvailableInPRInspect(t *testing.T) {
	registry := NewCommandRegistry()

	var dBinding *KeyBinding
	for _, binding := range registry.keyBindings {
		for _, key := range binding.Keys {
			if key == "d" {
				availableInPRInspect := false
				for _, state := range binding.AvailableIn {
					if state == ViewPRInspect {
						availableInPRInspect = true
						break
					}
				}
				if availableInPRInspect && binding.Description == "View diff" {
					dBinding = binding
					break
				}
			}
		}
	}

	if dBinding == nil {
		t.Error("expected 'd' key binding for 'View diff' to be available in ViewPRInspect")
	}
}

func TestHandleOpenBrowserKey_InPRInspect_WorksInBothModes(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect

	pr := &domain.PullRequest{
		ID:     "test-pr",
		Number: 42,
		Title:  "Test PR",
		URL:    "https://github.com/owner/repo/pull/42",
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
	}

	m.prInspect.SetPR(pr)

	m.prInspect.SwitchToDescription()
	_, cmd := handleOpenBrowserKey(m)

	if cmd != nil {
		t.Error("expected no command to be returned (browser opens synchronously)")
	}

	m.prInspect.SwitchToDiff()
	_, cmd2 := handleOpenBrowserKey(m)

	if cmd2 != nil {
		t.Error("expected no command to be returned (browser opens synchronously)")
	}
}

func TestCtrlOKeyBinding_AvailableInPRInspect(t *testing.T) {
	registry := NewCommandRegistry()

	var ctrlOBinding *KeyBinding
	for _, binding := range registry.keyBindings {
		for _, key := range binding.Keys {
			if key == "ctrl+o" {
				availableInPRInspect := false
				for _, state := range binding.AvailableIn {
					if state == ViewPRInspect {
						availableInPRInspect = true
						break
					}
				}
				if availableInPRInspect && binding.Description == "Open PR in browser" {
					ctrlOBinding = binding
					break
				}
			}
		}
	}

	if ctrlOBinding == nil {
		t.Error("expected 'ctrl+o' key binding for 'Open PR in browser' to be available in ViewPRInspect")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHandleInlineCommentKey_OnlyWorksInDiffMode(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{
						{Type: "add", Content: "+line1", NewLine: 1},
					}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SwitchToDescription()

	newModel, _ := handleInlineCommentKey(m)

	if newModel.inlineCommentView.IsActive() {
		t.Error("expected inline comment view to not activate in description mode")
	}

	m.prInspect.SwitchToDiff()
	newModel, _ = handleInlineCommentKey(m)

	if !newModel.inlineCommentView.IsActive() {
		t.Error("expected inline comment view to activate in diff mode")
	}
}

func TestIKeyBinding_AvailableInPRInspect(t *testing.T) {
	registry := NewCommandRegistry()

	var iBinding *KeyBinding
	for _, binding := range registry.keyBindings {
		for _, key := range binding.Keys {
			if key == "i" {
				availableInPRInspect := false
				for _, state := range binding.AvailableIn {
					if state == ViewPRInspect {
						availableInPRInspect = true
						break
					}
				}
				if availableInPRInspect && binding.Description == "Inline comment on line" {
					iBinding = binding
					break
				}
			}
		}
	}

	if iBinding == nil {
		t.Error("expected 'i' key binding for 'Inline comment on line' to be available in ViewPRInspect")
	}
}

func TestPendingComments_AddAndClear(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{
						{Type: "add", Content: "+line1", NewLine: 1},
					}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SwitchToDiff()

	if m.prInspect.GetPendingCommentCount() != 0 {
		t.Errorf("expected 0 pending comments initially, got %d", m.prInspect.GetPendingCommentCount())
	}

	m.prInspect.AddPendingComment("This is a test comment")

	if m.prInspect.GetPendingCommentCount() != 1 {
		t.Errorf("expected 1 pending comment after adding, got %d", m.prInspect.GetPendingCommentCount())
	}

	comments := m.prInspect.GetPendingComments()
	if len(comments) != 1 {
		t.Errorf("expected 1 pending comment in GetPendingComments(), got %d", len(comments))
	}

	if comments[0].Body != "This is a test comment" {
		t.Errorf("expected comment body to be 'This is a test comment', got '%s'", comments[0].Body)
	}

	if comments[0].FilePath != "file1.go" {
		t.Errorf("expected comment file path to be 'file1.go', got '%s'", comments[0].FilePath)
	}

	if comments[0].Line != 1 {
		t.Errorf("expected comment line to be 1, got %d", comments[0].Line)
	}

	m.prInspect.ClearPendingComments()

	if m.prInspect.GetPendingCommentCount() != 0 {
		t.Errorf("expected 0 pending comments after clearing, got %d", m.prInspect.GetPendingCommentCount())
	}
}

func TestLineNavigation_JKKeys(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect
	m.prInspect.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,3 +1,3 @@", Lines: []domain.DiffLine{
						{Type: "context", Content: " line1", NewLine: 1},
						{Type: "add", Content: "+line2", NewLine: 2},
						{Type: "delete", Content: "-line3", OldLine: 3},
					}},
				},
			},
		},
	}
	m.prInspect.SetDiff(diff)
	m.prInspect.SwitchToDiff()

	initialLine := m.prInspect.GetCurrentLineInfo()
	if initialLine == nil {
		t.Fatal("expected current line info to not be nil")
	}

	if initialLine.Content != " line1" {
		t.Errorf("expected to start on line 1, got '%s'", initialLine.Content)
	}

	m.prInspect.NextLine()
	nextLine := m.prInspect.GetCurrentLineInfo()
	if nextLine == nil {
		t.Fatal("expected current line info to not be nil after NextLine")
	}
	if nextLine.Content != "+line2" {
		t.Errorf("expected to be on line 2 after NextLine, got '%s'", nextLine.Content)
	}

	m.prInspect.PrevLine()
	prevLine := m.prInspect.GetCurrentLineInfo()
	if prevLine == nil {
		t.Fatal("expected current line info to not be nil after PrevLine")
	}
	if prevLine.Content != " line1" {
		t.Errorf("expected to be back on line 1 after PrevLine, got '%s'", prevLine.Content)
	}
}

func TestAKeyBinding_AvailableInPRInspect(t *testing.T) {
	registry := NewCommandRegistry()

	var aBinding *KeyBinding
	for _, binding := range registry.keyBindings {
		for _, key := range binding.Keys {
			if key == "a" {
				availableInPRInspect := false
				for _, state := range binding.AvailableIn {
					if state == ViewPRInspect {
						availableInPRInspect = true
						break
					}
				}
				if availableInPRInspect && binding.Description == "Approve PR" {
					aBinding = binding
					break
				}
			}
		}
	}

	if aBinding == nil {
		t.Error("expected 'a' key binding for 'Approve PR' to be available in ViewPRInspect")
	}
}

func TestRKeyBinding_AvailableInPRInspect(t *testing.T) {
	registry := NewCommandRegistry()

	var rBinding *KeyBinding
	for _, binding := range registry.keyBindings {
		for _, key := range binding.Keys {
			if key == "r" {
				availableInPRInspect := false
				for _, state := range binding.AvailableIn {
					if state == ViewPRInspect {
						availableInPRInspect = true
						break
					}
				}
				if availableInPRInspect && binding.Description == "Request changes" {
					rBinding = binding
					break
				}
			}
		}
	}

	if rBinding == nil {
		t.Error("expected 'r' key binding for 'Request changes' to be available in ViewPRInspect")
	}
}

func TestHandleApproveKey_NotInPRInspect_DoesNothing(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList

	m, _ = handleApproveKey(m)

	if m.reviewView.IsActive() {
		t.Error("expected review view to not activate when not in PR inspect view")
	}
}

func TestHandleRequestChangesKey_NotInPRInspect_DoesNothing(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList

	m, _ = handleRequestChangesKey(m)

	if m.reviewView.IsActive() {
		t.Error("expected review view to not activate when not in PR inspect view")
	}
}

func TestHandleApproveKey_ActivatesReviewModeApprove(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect

	m, _ = handleApproveKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to be active")
	}

	review := m.reviewView.GetReview()
	if review.Action != domain.ReviewActionApprove {
		t.Errorf("expected review action to be approve, got %s", review.Action)
	}
}

func TestHandleRequestChangesKey_ActivatesReviewModeRequestChanges(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect

	m, _ = handleRequestChangesKey(m)

	if !m.reviewView.IsActive() {
		t.Error("expected review view to be active")
	}

	review := m.reviewView.GetReview()
	if review.Action != domain.ReviewActionRequestChanges {
		t.Errorf("expected review action to be request_changes, got %s", review.Action)
	}
}

func TestReviewView_GetReview_IncludesBody(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRInspect

	m, _ = handleApproveKey(m)

	review := m.reviewView.GetReview()

	if review.Action != domain.ReviewActionApprove {
		t.Errorf("expected review action to be approve, got %s", review.Action)
	}

	if review.Comments == nil {
		t.Error("expected review comments to not be nil")
	}
}
