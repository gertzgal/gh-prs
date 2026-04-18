package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/render"
)

type stubClient struct {
	repo *model.Repo
	err  error
}

func (c stubClient) FetchRepo(_ context.Context) (*model.Repo, error) {
	return c.repo, c.err
}

type stubFormatter struct {
	label       string
	panicOnCall bool
	lastCtx     render.Context
	calls       int
}

func (f *stubFormatter) Format(_ *model.Repo, ctx render.Context) string {
	if f.panicOnCall {
		panic(fmt.Sprintf("%s formatter should not have been invoked", f.label))
	}
	f.calls++
	f.lastCtx = ctx
	return fmt.Sprintf("%s:latency=%d", f.label, ctx.LatencyMs)
}

var samplePR = model.PR{
	Number: 1, Title: "Test PR",
	URL:         "https://github.com/acme/widget/pull/1",
	HeadRefName: "feat/test", BaseRefName: "main",
	Additions: 10, Deletions: 2, ChangedFiles: 3,
	CiState:          model.CiSuccess,
	MergeStateStatus: "CLEAN",
}

func baseRepo() *model.Repo {
	return &model.Repo{
		Owner: "acme", Name: "widget",
		DefaultBranch: "main", ViewerLogin: "alice",
		PRs: []model.PR{samplePR},
	}
}

func emptyRepo() *model.Repo {
	r := baseRepo()
	r.PRs = nil
	return r
}

// tickingNow returns a now() that yields t0 on first call and t0+dur on second.
func tickingNow(dur time.Duration) func() time.Time {
	t0 := time.Unix(0, 0)
	calls := 0
	return func() time.Time {
		t := t0.Add(time.Duration(calls) * dur)
		calls++
		return t
	}
}

func newHarness(overrides func(*Deps)) (Deps, *bytes.Buffer, *bytes.Buffer, *stubFormatter) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	text := &stubFormatter{label: "TEXT"}
	d := Deps{
		Flags:     Flags{},
		Client:    stubClient{repo: baseRepo()},
		Formatter: text,
		FormatCtx: render.Context{},
		Stdout:    stdout,
		Stderr:    stderr,
		Now:       tickingNow(300 * time.Millisecond),
	}
	if overrides != nil {
		overrides(&d)
	}
	return d, stdout, stderr, text
}

func TestRun_HappyPathText(t *testing.T) {
	d, stdout, stderr, text := newHarness(nil)
	code := Run(context.Background(), d)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if text.calls != 1 {
		t.Fatalf("formatter calls = %d, want 1", text.calls)
	}
	if got := stdout.String(); got != "TEXT:latency=300" {
		t.Fatalf("stdout = %q, want %q", got, "TEXT:latency=300")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRun_HappyPathMachine(t *testing.T) {
	jsonF := &stubFormatter{label: "JSON"}
	d, stdout, _, _ := newHarness(func(d *Deps) {
		d.Flags.Machine = true
		d.Formatter = jsonF
	})
	code := Run(context.Background(), d)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(stdout.String(), "JSON:latency=") {
		t.Fatalf("stdout missing JSON formatter output: %q", stdout.String())
	}
}

func TestRun_EmptyPRsText(t *testing.T) {
	text := &stubFormatter{label: "TEXT", panicOnCall: true}
	d, stdout, _, _ := newHarness(func(d *Deps) {
		d.Client = stubClient{repo: emptyRepo()}
		d.Formatter = text
	})
	code := Run(context.Background(), d)
	if code != exitNoPRs {
		t.Fatalf("exit = %d, want %d", code, exitNoPRs)
	}
	if !strings.Contains(stdout.String(), "No open PRs authored by @alice in acme/widget") {
		t.Fatalf("stdout missing friendly message: %q", stdout.String())
	}
}

func TestRun_EmptyPRsMachine(t *testing.T) {
	jsonF := &stubFormatter{label: "JSON"}
	d, stdout, _, _ := newHarness(func(d *Deps) {
		d.Flags.Machine = true
		d.Client = stubClient{repo: emptyRepo()}
		d.Formatter = jsonF
	})
	code := Run(context.Background(), d)
	if code != exitNoPRs {
		t.Fatalf("exit = %d, want %d", code, exitNoPRs)
	}
	if !strings.Contains(stdout.String(), "JSON:latency=") {
		t.Fatalf("stdout missing JSON formatter output: %q", stdout.String())
	}
	if jsonF.calls != 1 {
		t.Fatalf("json formatter calls = %d, want 1", jsonF.calls)
	}
}

func TestRun_RepoNotFound(t *testing.T) {
	d, _, stderr, _ := newHarness(func(d *Deps) {
		d.Client = stubClient{err: model.ErrRepoNotFound}
	})
	code := Run(context.Background(), d)
	if code != exitNoRepo {
		t.Fatalf("exit = %d, want %d", code, exitNoRepo)
	}
	if !strings.Contains(stderr.String(), "not inside a GitHub repo") {
		t.Fatalf("stderr missing message: %q", stderr.String())
	}
}

func TestRun_WrappedRepoNotFound(t *testing.T) {
	d, _, stderr, _ := newHarness(func(d *Deps) {
		d.Client = stubClient{err: fmt.Errorf("wrap: %w", model.ErrRepoNotFound)}
	})
	code := Run(context.Background(), d)
	if code != exitNoRepo {
		t.Fatalf("exit = %d, want %d", code, exitNoRepo)
	}
	if !strings.Contains(stderr.String(), "not inside a GitHub repo") {
		t.Fatalf("stderr missing message: %q", stderr.String())
	}
}

func TestRun_GhError(t *testing.T) {
	d, _, stderr, _ := newHarness(func(d *Deps) {
		d.Client = stubClient{err: &model.GhError{Msg: "bang", Stderr: "stderr body"}}
	})
	code := Run(context.Background(), d)
	if code != exitGhError {
		t.Fatalf("exit = %d, want %d", code, exitGhError)
	}
	out := stderr.String()
	if !strings.Contains(out, "gh prs: bang") {
		t.Errorf("stderr missing message: %q", out)
	}
	if !strings.Contains(out, "stderr body") {
		t.Errorf("stderr missing stderr body: %q", out)
	}
}

func TestRun_UnexpectedError(t *testing.T) {
	d, _, stderr, _ := newHarness(func(d *Deps) {
		d.Client = stubClient{err: errors.New("mystery")}
	})
	code := Run(context.Background(), d)
	if code != exitGhError {
		t.Fatalf("exit = %d, want %d", code, exitGhError)
	}
	if !strings.Contains(stderr.String(), "gh prs: unexpected error: mystery") {
		t.Fatalf("stderr unexpected: %q", stderr.String())
	}
}

func TestRun_LatencyInjectedIntoFormatter(t *testing.T) {
	text := &stubFormatter{label: "TEXT"}
	d, _, _, _ := newHarness(func(d *Deps) {
		d.Formatter = text
		d.Now = tickingNow(450 * time.Millisecond)
	})
	_ = Run(context.Background(), d)
	if text.lastCtx.LatencyMs != 450 {
		t.Fatalf("formatter got latency = %d, want 450", text.lastCtx.LatencyMs)
	}
}
