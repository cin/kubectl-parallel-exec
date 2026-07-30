# kubectl-parallel-exec

[![Release Workflow](https://github.com/cin/kubectl-parallel-exec/actions/workflows/release.yml/badge.svg)](https://github.com/cin/kubectl-parallel-exec/actions/workflows/release.yml)

![Project Logo](./assets/logo.png)

A Go CLI for executing commands in parallel on Kubernetes pods selected by labels, similar to GNU `parallel`. Originally scaffolded with GPT-4 assistance.

## Features

- Execute commands in parallel on pods matched by a label selector
- Target a specific container in each pod
- Cap concurrency with `-j` and set a per-pod `-timeout`
- Aggregate results sorted by pod name
- Non-zero exit if any pod exec fails

## Requirements

- Go 1.26 or higher (to build from source)
- A valid kubeconfig file or in-cluster configuration
- Access to a Kubernetes cluster

## Installation

### Homebrew

```bash
brew install cin/kubectl-parallel-exec/kubectl-parallel-exec
```

### From Source

```sh
git clone https://github.com/cin/kubectl-parallel-exec.git
cd kubectl-parallel-exec
go build -o kubectl-parallel-exec
sudo mv kubectl-parallel-exec /usr/local/bin/
```

With the binary named `kubectl-parallel-exec` on your `PATH`, kubectl plugins resolve it as `kubectl parallel exec`.

## Usage

```sh
kubectl-parallel-exec -c container-name -l label-selector [-n namespace] [-j 16] [-timeout 5m] [--] command args...
```

| Flag | Description |
|------|-------------|
| `-kubeconfig` | Path to kubeconfig. Falls back to `KUBECONFIG`, then in-cluster config. |
| `-c` | Container to exec into (required). |
| `-l` | Label selector (required). |
| `-n` | Namespace. Empty lists pods across all namespaces. |
| `-j` | Max concurrent execs (default `16`; `0` = unlimited). |
| `-timeout` | Per-pod exec timeout (default `0` = no timeout). |
| `-v` | Print the version. |

### Example

```sh
kubectl-parallel-exec -c cassandra -l app=cassandra -n cassandra-ns -j 8 nodetool status
```

If authentication is enabled and you don't want to expose credentials:

```bash
export KPE_OPTS="-l cassandra-cluster-component=cassandra,cassandra-cluster-instance=test-cluster"
alias kpe="kubectl parallel exec $KPE_OPTS"

function knt() {
  local nodetool_command="${*}"
  local nodetool_auth_opts="--ssl -u \$(cat /etc/cassandra-auth-config/admin-role) -pw \$(cat /etc/cassandra-auth-config/admin-password)"
  kpe -c cassandra -- bash -c "nodetool ${nodetool_auth_opts} ${nodetool_command}"
}
```

Modify `KPE_OPTS` for the target pod set; `KUBECONFIG` selects the cluster.

## Credits

The initial implementation was largely provided by ChatGPT. Subsequent modernization and hardening were done by hand.
