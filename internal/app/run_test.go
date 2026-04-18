package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/render"
)

type stubClient struct {
	repo        *model.Repo
	err         error
	lastFilters filter.Set
}

func (c *stubClient) FetchRepo(_ context.Context, filters filter.Set) (*model.Repo, error) {
	c.lastFilters = filters
	return c.repo, c.err
}

type stubFormatter struct {
	label       string
	panicOnCall bool
	err         error
	lastCtx     render.Context
	calls       int
}

func (f *stubFormatter) Format(_ *model.Repo, ctx render.Context) (string, error) {
	if f.panicOnCall {
		panic(fmt.Sprintf("%s formatter should not have been invoked", f.label))
	}
	f.calls++
	f.lastCtx = ctx
	if f.err != nil {
		return "", f.err
	}
	return fmt.Sprintf("%s:latency=%d", f.label, ctx.LatencyMs), nil
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
		Filters:   filter.Set{},
		Client:    &stubClient{repo: baseRepo()},
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
		d.Client = &stubClient{repo: emptyRepo()}
		d.Formatter = text
	})
	code := Run(context.Background(), d)
	if code != exitNoPRs {
		t.Fatalf("exit = %d, want %d", code, exitNoPRs)
	}
	if !strings.Contains(stdout.String(), "No open PRs matching the applied filters in acme/widget") {
		t.Fatalf("stdout missing friendly message: %q", stdout.String())
	}
}

func TestRun_EmptyPRsMachine(t *testing.T) {
	jsonF := &stubFormatter{label: "JSON"}
	d, stdout, _, _ := newHarness(func(d *Deps) {
		d.Flags.Machine = true
		d.Client = &stubClient{repo: emptyRepo()}
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
		d.Client = &stubClient{err: model.ErrRepoNotFound}
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
		d.Client = &stubClient{err: fmt.Errorf("wrap: %w", model.ErrRepoNotFound)}
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
		d.Client = &stubClient{err: &model.GhError{Msg: "bang", Stderr: "stderr body"}}
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
		d.Client = &stubClient{err: errors.New("mystery")}
	})
	code := Run(context.Background(), d)
	if code != exitGhError {
		t.Fatalf("exit = %d, want %d", code, exitGhError)
	}
	if !strings.Contains(stderr.String(), "gh prs: unexpected error: mystery") {
		t.Fatalf("stderr unexpected: %q", stderr.String())
	}
}

func TestRun_FormatErrorSurfacedToStderr(t *testing.T) {
	boom := errors.New("encoder boom")
	text := &stubFormatter{label: "TEXT", err: boom}
	d, stdout, stderr, _ := newHarness(func(d *Deps) {
		d.Formatter = text
	})
	code := Run(context.Background(), d)
	if code != exitGhError {
		t.Fatalf("exit = %d, want %d", code, exitGhError)
	}
	if !strings.Contains(stderr.String(), "gh prs: format: encoder boom") {
		t.Fatalf("stderr missing format error: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty on format error, got %q", stdout.String())
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

// ---------------------------------------------------------------------------
// Filter wiring tests
// ---------------------------------------------------------------------------

func TestRun_FiltersPassedToClient(t *testing.T) {
	sc := &stubClient{repo: baseRepo()}
	filters := filter.NewSet(
		[]filter.QueryFilter{filter.NewAuthorFilter([]string{"alice"})},
		nil,
	)
	d, _, _, _ := newHarness(func(d *Deps) {
		d.Client = sc
		d.Filters = filters
	})
	_ = Run(context.Background(), d)

	// Verify the exact filter set the client received.
	got := sc.lastFilters.QueryFragments()
	if len(got) != 1 || got[0] != "author:alice" {
		t.Fatalf("client received fragments %v, want [author:alice]", got)
	}
}

// stubListFilter keeps only PRs whose Number is in the allow-list.
type stubListFilter struct{ allow []int }

func (f stubListFilter) Apply(prs []model.PR) []model.PR {
	var out []model.PR
	for _, pr := range prs {
		for _, n := range f.allow {
			if pr.Number == n {
				out = append(out, pr)
				break
			}
		}
	}
	return out
}

func TestRun_ListFilterApplied_ReducesPRs(t *testing.T) {
	// baseRepo() has samplePR (#1). A list filter that allows only PR #99
	// should produce an empty slice → exitNoPRs.
	lf := stubListFilter{allow: []int{99}}
	filters := filter.NewSet(nil, []filter.ListFilter{lf})
	text := &stubFormatter{label: "TEXT", panicOnCall: true}

	d, stdout, _, _ := newHarness(func(d *Deps) {
		d.Filters = filters
		d.Formatter = text
	})
	code := Run(context.Background(), d)
	if code != exitNoPRs {
		t.Fatalf("exit = %d, want exitNoPRs (%d)", code, exitNoPRs)
	}
	if !strings.Contains(stdout.String(), "No open PRs matching") {
		t.Fatalf("stdout missing no-PRs message: %q", stdout.String())
	}
}

func TestRun_ListFilterApplied_PassthroughWhenAllowed(t *testing.T) {
	// Allow PR #1 (the samplePR) → formatter is called, returns success.
	lf := stubListFilter{allow: []int{1}}
	filters := filter.NewSet(nil, []filter.ListFilter{lf})

	d, _, _, text := newHarness(func(d *Deps) {
		d.Filters = filters
	})
	code := Run(context.Background(), d)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if text.calls != 1 {
		t.Fatalf("formatter calls = %d, want 1", text.calls)
	}
}
