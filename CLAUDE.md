# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`kubectl-parallel-exec` is a Go CLI (usable as a kubectl plugin, invoked as `kubectl parallel exec`) that runs a
command in parallel across all pods matching a label selector, similar to GNU `parallel`. The entire implementation
is in `main.go`; tests are in `main_test.go`. There are no other packages.

## Commands

```sh
go build -o kubectl-parallel-exec   # build
go test -race ./...                 # run all tests (CI always uses -race)
go test -race -run TestName ./...   # run a single test
golangci-lint run --timeout=5m      # lint (CI pins golangci-lint-action v2.12.2; no repo-local config file)
```

There is no Makefile; use the `go` toolchain directly. CI (`.github/workflows/pull_request.yml`) runs lint, test,
and a `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` build on every PR into `main`.

## Architecture

Execution flow in `main.go`, in order:

1. **Flag parsing / validation** (`main`) — `-c` (container) and `-l` (label selector) are required; `-j`
   (concurrency, default 16, `0` = unlimited) and `-timeout` (per-pod, `0` = none) gate behavior downstream.
2. **Kubeconfig resolution** (`selectKubeconfig`) — `-kubeconfig` flag wins, then `$KUBECONFIG`, then falls back to
   `rest.InClusterConfig()` if out-of-cluster config building fails.
3. **Client throughput tuning** (`tuneClientThroughput`) — sets `rest.Config.QPS`/`Burst` from `-j` so the
   client-go rate limiter doesn't throttle below the requested concurrency (independent of the semaphore below).
4. **Pod listing** — one `List` call with the label selector; `-n` empty means all namespaces.
5. **Parallel exec** (`runParallelExec`) — fans out one goroutine per pod, gated by a semaphore channel sized to
   `-j` (or unbounded if `0`), collects `PodResult` per pod over a buffered channel, then `sortPodResults` by pod
   name for deterministic output.
6. **Per-pod exec** (`execCommand` / `newExecutor`) — builds the pods/exec subresource request, then executes via
   `remotecommand.NewFallbackExecutor(websocketExec, spdyExec, ...)`: prefers WebSocket, falls back to SPDY on
   upgrade/proxy failures. Stdout and stderr are captured into separate buffers and concatenated by
   `combineOutput` — note this is **not** interleaved in real arrival order the way `kubectl exec` output is.
7. **Reporting** — `formatPodResult` prints a colored per-pod header (pod name, elapsed time) followed by output;
   any pod exec error is reported inline per-pod (not fatal), but causes a non-zero process exit at the end.

Key invariant: a failure on one pod never aborts others — all pods are attempted and results are aggregated before
the process decides its exit code.

## Release process

Releases are cut via GitHub Releases (`release.yml`): cross-compiles linux/darwin amd64/arm64 binaries with
`-ldflags "-X main.version=<tag>"`, uploads tarballs to the release, and pushes an updated formula (with recomputed
sha256s) to the separate `cin/homebrew-kubectl-parallel-exec` tap repo.
