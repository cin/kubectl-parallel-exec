package main

import (
	"errors"
	"sort"
	"strings"
	"testing"
	"time"
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

func TestByPodNameSortsResultsByPodName(t *testing.T) {
	results := []PodResult{
		{podName: "pod-c"},
		{podName: "pod-a"},
		{podName: "pod-b"},
	}

	sort.Sort(ByPodName(results))

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
