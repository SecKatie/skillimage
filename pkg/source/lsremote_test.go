package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLsRemoteReturnsCommitSHA(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip("git not available")
	}
	repo := newRemoteTestRepo(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sha, err := LsRemote(ctx, repo, "HEAD")
	if err != nil {
		t.Fatalf("LsRemote: %v", err)
	}
	if len(sha) < 7 {
		t.Errorf("expected commit SHA, got %q", sha)
	}
	if !commitSHAPattern.MatchString(sha) {
		t.Errorf("SHA %q does not match commit pattern", sha)
	}
}

func TestLsRemoteBadRef(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip("git not available")
	}
	repo := newRemoteTestRepo(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := LsRemote(ctx, repo, "nonexistent-branch-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent ref")
	}
}

func newRemoteTestRepo(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	bareDir := filepath.Join(t.TempDir(), "repo.git")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	run(workDir, "init")
	run(workDir, "config", "user.email", "test@test.com")
	run(workDir, "config", "user.name", "Test")
	run(workDir, "config", "commit.gpgSign", "false")
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}
	run(workDir, "add", ".")
	run(workDir, "commit", "-m", "init")
	run(workDir, "clone", "--bare", ".", bareDir)
	return bareDir
}
