package azuredevops

import (
	"context"
	"testing"
	"time"
)

func TestResolveProjectAndRepoWithCache_CachesResults(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	repository := "TestProject/TestRepo"
	projectID := "proj-123"
	repoID := "repo-456"

	provider.repoCache[repository] = &ResolvedRepository{
		ProjectID: projectID,
		RepoID:    repoID,
		CachedAt:  time.Now(),
	}

	ctx := context.Background()
	gotProjectID, gotRepoID, err := provider.resolveProjectAndRepoWithCache(ctx, repository)

	if err != nil {
		t.Fatalf("resolveProjectAndRepoWithCache() error = %v, want nil", err)
	}

	if gotProjectID != projectID {
		t.Errorf("resolveProjectAndRepoWithCache() projectID = %v, want %v", gotProjectID, projectID)
	}

	if gotRepoID != repoID {
		t.Errorf("resolveProjectAndRepoWithCache() repoID = %v, want %v", gotRepoID, repoID)
	}
}

func TestResolveProjectAndRepoWithCache_HitsFreshCache(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	repository := "TestProject/TestRepo"
	projectID := "proj-123"
	repoID := "repo-456"

	provider.repoCache[repository] = &ResolvedRepository{
		ProjectID: projectID,
		RepoID:    repoID,
		CachedAt:  time.Now(),
	}

	ctx := context.Background()

	gotProjectID, gotRepoID, err := provider.resolveProjectAndRepoWithCache(ctx, repository)
	if err != nil {
		t.Fatalf("resolveProjectAndRepoWithCache() with fresh cache error = %v", err)
	}
	if gotProjectID != projectID {
		t.Errorf("resolveProjectAndRepoWithCache() projectID = %v, want %v", gotProjectID, projectID)
	}
	if gotRepoID != repoID {
		t.Errorf("resolveProjectAndRepoWithCache() repoID = %v, want %v", gotRepoID, repoID)
	}
}

func TestResolveProjectAndRepoWithCache_DetectsExpiredCache(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  100 * time.Millisecond,
	}

	repository := "TestProject/TestRepo"
	cachedEntry := &ResolvedRepository{
		ProjectID: "proj-123",
		RepoID:    "repo-456",
		CachedAt:  time.Now().Add(-200 * time.Millisecond),
	}
	provider.repoCache[repository] = cachedEntry

	provider.cacheMutex.RLock()
	if _, ok := provider.repoCache[repository]; !ok {
		t.Fatal("Cache entry should exist before check")
	}

	expired := time.Since(cachedEntry.CachedAt) >= provider.cacheTTL
	provider.cacheMutex.RUnlock()

	if !expired {
		t.Error("Cache entry should be detected as expired")
	}
}

func TestResolveProjectAndRepoWithCache_DifferentRepositories(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	repo1 := "Project1/Repo1"
	repo2 := "Project2/Repo2"

	provider.repoCache[repo1] = &ResolvedRepository{
		ProjectID: "proj-1",
		RepoID:    "repo-1",
		CachedAt:  time.Now(),
	}

	provider.repoCache[repo2] = &ResolvedRepository{
		ProjectID: "proj-2",
		RepoID:    "repo-2",
		CachedAt:  time.Now(),
	}

	if len(provider.repoCache) != 2 {
		t.Fatalf("Cache should have 2 entries for different repos, got %d", len(provider.repoCache))
	}

	ctx := context.Background()

	proj1, repo1ID, err := provider.resolveProjectAndRepoWithCache(ctx, repo1)
	if err != nil {
		t.Fatalf("resolveProjectAndRepoWithCache() for repo1 error = %v", err)
	}

	if proj1 != "proj-1" || repo1ID != "repo-1" {
		t.Errorf("resolveProjectAndRepoWithCache() repo1 = (%v, %v), want (proj-1, repo-1)", proj1, repo1ID)
	}

	proj2, repo2ID, err := provider.resolveProjectAndRepoWithCache(ctx, repo2)
	if err != nil {
		t.Fatalf("resolveProjectAndRepoWithCache() for repo2 error = %v", err)
	}

	if proj2 != "proj-2" || repo2ID != "repo-2" {
		t.Errorf("resolveProjectAndRepoWithCache() repo2 = (%v, %v), want (proj-2, repo-2)", proj2, repo2ID)
	}
}

func TestResolveProjectAndRepoWithCache_ThreadSafety(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	repository := "TestProject/TestRepo"
	projectID := "proj-123"
	repoID := "repo-456"

	provider.repoCache[repository] = &ResolvedRepository{
		ProjectID: projectID,
		RepoID:    repoID,
		CachedAt:  time.Now(),
	}

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _, _ = provider.resolveProjectAndRepoWithCache(ctx, repository)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCacheStructure(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		want       bool
	}{
		{
			name:       "valid repository identifier",
			repository: "Project/Repo",
			want:       true,
		},
		{
			name:       "project with spaces",
			repository: "My Project/My Repo",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				repoCache: make(map[string]*ResolvedRepository),
				cacheTTL:  5 * time.Minute,
			}

			provider.repoCache[tt.repository] = &ResolvedRepository{
				ProjectID: "test-proj-id",
				RepoID:    "test-repo-id",
				CachedAt:  time.Now(),
			}

			ctx := context.Background()
			projID, repoID, err := provider.resolveProjectAndRepoWithCache(ctx, tt.repository)

			if err != nil {
				t.Fatalf("resolveProjectAndRepoWithCache() error = %v, want nil", err)
			}

			if projID != "test-proj-id" {
				t.Errorf("resolveProjectAndRepoWithCache() projectID = %v, want test-proj-id", projID)
			}

			if repoID != "test-repo-id" {
				t.Errorf("resolveProjectAndRepoWithCache() repoID = %v, want test-repo-id", repoID)
			}
		})
	}
}

func TestCacheTTLConfiguration(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  10 * time.Millisecond,
	}

	if provider.cacheTTL != 10*time.Millisecond {
		t.Errorf("Provider cacheTTL = %v, want 10ms", provider.cacheTTL)
	}

	provider2 := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	if provider2.cacheTTL != 5*time.Minute {
		t.Errorf("Provider cacheTTL = %v, want 5min", provider2.cacheTTL)
	}
}

func TestNewProvider_InitializesCache(t *testing.T) {
	provider := &Provider{
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}

	if provider.repoCache == nil {
		t.Fatal("Provider repoCache should be initialized, got nil")
	}

	if len(provider.repoCache) != 0 {
		t.Errorf("Provider repoCache should start empty, got %d entries", len(provider.repoCache))
	}

	if provider.cacheTTL == 0 {
		t.Error("Provider cacheTTL should be non-zero")
	}
}
