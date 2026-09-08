package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestSelectKubeconfig(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envValue  string
		want      string
	}{
		{
			name:      "flag value wins",
			flagValue: "/tmp/explicit-config",
			envValue:  "/tmp/env-config",
			want:      "/tmp/explicit-config",
		},
		{
			name:      "env value is fallback",
			flagValue: "",
			envValue:  "/tmp/env-config",
			want:      "/tmp/env-config",
		},
		{
			name:      "empty when neither provided",
			flagValue: "",
			envValue:  "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectKubeconfig(tt.flagValue, tt.envValue); got != tt.want {
				t.Fatalf("selectKubeconfig() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterRunningPods(t *testing.T) {
	pod := func(name string, phase v1.PodPhase) v1.Pod {
		return v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Status:     v1.PodStatus{Phase: phase},
		}
	}

	pods := []v1.Pod{
		pod("running-1", v1.PodRunning),
		pod("succeeded-1", v1.PodSucceeded),
		pod("running-2", v1.PodRunning),
		pod("pending-1", v1.PodPending),
	}

	running, skipped := filterRunningPods(pods)

	gotRunning := []string{running[0].Name, running[1].Name}
	wantRunning := []string{"running-1", "running-2"}
	for i := range wantRunning {
		if gotRunning[i] != wantRunning[i] {
			t.Fatalf("running pod names = %v, want %v", gotRunning, wantRunning)
		}
	}

	gotSkipped := []string{skipped[0].Name, skipped[1].Name}
	wantSkipped := []string{"succeeded-1", "pending-1"}
	for i := range wantSkipped {
		if gotSkipped[i] != wantSkipped[i] {
			t.Fatalf("skipped pod names = %v, want %v", gotSkipped, wantSkipped)
		}
	}
}

func TestRunParallelExecReturnsResultForEachPod(t *testing.T) {
	pods := []v1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns-a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-b", Namespace: "ns-b"}},
	}

	results, interrupted := runParallelExec(context.Background(), pods, 2, func(_ context.Context, podName, namespace string) PodResult {
		return PodResult{podName: podName, output: namespace}
	})

	if interrupted {
		t.Fatal("interrupted = true, want false")
	}

	if len(results) != len(pods) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(pods))
	}

	got := make(map[string]string, len(results))
	for _, r := range results {
		got[r.podName] = r.output
	}
	if got["pod-a"] != "ns-a" || got["pod-b"] != "ns-b" {
		t.Fatalf("results = %v, want pod-a:ns-a and pod-b:ns-b", got)
	}
}

func TestRunParallelExecEmptyPods(t *testing.T) {
	results, interrupted := runParallelExec(context.Background(), nil, 4, func(context.Context, string, string) PodResult {
		t.Fatal("exec should not be called with no pods")
		return PodResult{}
	})
	if results != nil {
		t.Fatalf("runParallelExec(nil pods) = %v, want nil", results)
	}
	if interrupted {
		t.Fatal("interrupted = true, want false")
	}
}

func TestRunParallelExecStopsOnContextCancellation(t *testing.T) {
	pods := []v1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-0"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, len(pods))
	exec := func(ctx context.Context, podName, namespace string) PodResult {
		started <- struct{}{}
		<-ctx.Done()
		return PodResult{podName: podName}
	}

	type outcome struct {
		results     []PodResult
		interrupted bool
	}
	done := make(chan outcome, 1)
	go func() {
		results, interrupted := runParallelExec(ctx, pods, 2, exec)
		done <- outcome{results, interrupted}
	}()

	for i := 0; i < len(pods); i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d/%d execs started before timeout", i, len(pods))
		}
	}

	cancel()

	select {
	case got := <-done:
		if !got.interrupted {
			t.Fatal("interrupted = false, want true after ctx cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runParallelExec did not return after context cancellation")
	}
}

func TestRunParallelExecHonorsConcurrencyLimit(t *testing.T) {
	pods := []v1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-0"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-2"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-3"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-4"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-5"}},
	}
	const limit = 2

	started := make(chan struct{}, len(pods))
	release := make(chan struct{})
	exec := func(_ context.Context, podName, namespace string) PodResult {
		started <- struct{}{}
		<-release
		return PodResult{podName: podName}
	}

	done := make(chan []PodResult, 1)
	go func() {
		results, _ := runParallelExec(context.Background(), pods, limit, exec)
		done <- results
	}()

	for i := 0; i < limit; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d/%d execs started before timeout, want %d to run concurrently", i, limit, limit)
		}
	}

	select {
	case <-started:
		t.Fatalf("more than %d execs started concurrently, limit was not honored", limit)
	case <-time.After(100 * time.Millisecond):
		// no additional exec started while at the limit, as expected
	}

	close(release)

	select {
	case results := <-done:
		if len(results) != len(pods) {
			t.Fatalf("len(results) = %d, want %d", len(results), len(pods))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runParallelExec did not return after release")
	}
}

func TestRunParallelExecZeroConcurrencyRunsAllPodsAtOnce(t *testing.T) {
	pods := []v1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-0"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-2"}},
	}

	started := make(chan struct{}, len(pods))
	release := make(chan struct{})
	exec := func(_ context.Context, podName, namespace string) PodResult {
		started <- struct{}{}
		<-release
		return PodResult{podName: podName}
	}

	done := make(chan []PodResult, 1)
	go func() {
		results, _ := runParallelExec(context.Background(), pods, 0, exec)
		done <- results
	}()

	for i := 0; i < len(pods); i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d/%d execs started concurrently before timeout, want all pods to run at once", i, len(pods))
		}
	}

	close(release)

	select {
	case results := <-done:
		if len(results) != len(pods) {
			t.Fatalf("len(results) = %d, want %d", len(results), len(pods))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runParallelExec did not return after release")
	}
}

func TestTranslateExecErrorNamesTimeoutFlag(t *testing.T) {
	err := translateExecError(context.DeadlineExceeded, 5*time.Second)

	for _, want := range []string{"-timeout", "5s"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("translateExecError() = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestTranslateExecErrorPassesThroughOtherErrors(t *testing.T) {
	original := errors.New("boom")
	if got := translateExecError(original, time.Second); got != original {
		t.Fatalf("translateExecError() = %v, want unchanged %v", got, original)
	}
}

func TestSortPodResultsByName(t *testing.T) {
	results := []PodResult{
		{podName: "pod-c"},
		{podName: "pod-a"},
		{podName: "pod-b"},
	}

	sortPodResults(results)

	got := []string{results[0].podName, results[1].podName, results[2].podName}
	want := []string{"pod-a", "pod-b", "pod-c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted pod names = %v, want %v", got, want)
		}
	}
}

func TestFormatPodResultIncludesResultError(t *testing.T) {
	output := formatPodResult(PodResult{
		podName: "pod-a",
		output:  "partial output\n",
		err:     errors.New("command failed"),
		elapsed: time.Second,
	})

	for _, want := range []string{"pod-a", "command failed", "partial output"} {
		if !strings.Contains(output, want) {
			t.Fatalf("formatPodResult() = %q, want it to contain %q", output, want)
		}
	}

	if strings.Contains(output, "<nil>") {
		t.Fatalf("formatPodResult() = %q, did not expect nil error output", output)
	}
}

func TestFormatPodResultOmitsErrorPrefixOnSuccess(t *testing.T) {
	output := formatPodResult(PodResult{
		podName: "pod-a",
		output:  "command output\n",
		elapsed: time.Second,
	})

	if strings.Contains(output, "Error executing command") {
		t.Fatalf("formatPodResult() = %q, did not expect error prefix", output)
	}

	if !strings.Contains(output, "command output") {
		t.Fatalf("formatPodResult() = %q, want command output", output)
	}
}

func TestFormatPodResultAddsTrailingNewline(t *testing.T) {
	output := formatPodResult(PodResult{
		podName: "pod-a",
		output:  "no trailing newline",
		elapsed: time.Second,
	})

	if !strings.HasSuffix(output, "no trailing newline\n") {
		t.Fatalf("formatPodResult() = %q, want trailing newline appended", output)
	}
}

func TestCombineOutputMergesStdoutAndStderr(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{name: "stdout only", stdout: "out\n", stderr: "", want: "out\n"},
		{name: "stderr only", stdout: "", stderr: "err\n", want: "err\n"},
		{name: "both", stdout: "out\n", stderr: "err\n", want: "out\nerr\n"},
		{name: "empty", stdout: "", stderr: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := combineOutput(tt.stdout, tt.stderr); got != tt.want {
				t.Fatalf("combineOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTuneClientThroughput(t *testing.T) {
	cfg := &rest.Config{}
	tuneClientThroughput(cfg, 8)
	if cfg.QPS != 8 {
		t.Fatalf("QPS = %v, want 8", cfg.QPS)
	}
	if cfg.Burst != 16 {
		t.Fatalf("Burst = %v, want 16", cfg.Burst)
	}

	cfg = &rest.Config{}
	tuneClientThroughput(cfg, 0)
	if cfg.RateLimiter == nil {
		t.Fatal("RateLimiter = nil, want an unthrottled limiter for -j 0")
	}
	if !cfg.RateLimiter.TryAccept() {
		t.Fatal("RateLimiter.TryAccept() = false, want -j 0 to never throttle")
	}
}
