package main

import (
	"errors"
	"strings"
	"testing"
	"time"

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
	if cfg.QPS != float32(defaultConcurrency) {
		t.Fatalf("QPS = %v, want %v", cfg.QPS, defaultConcurrency)
	}
	if cfg.Burst != defaultConcurrency*2 {
		t.Fatalf("Burst = %v, want %v", cfg.Burst, defaultConcurrency*2)
	}
}
