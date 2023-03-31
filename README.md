![Project Logo](./assets/logo.png)
# kubectl-parallel-exec

[![Release Workflow](https://github.com/cin/kubectl-parallel-exec/actions/workflows/release.yml/badge.svg)](https://github.com/cin/kubectl-parallel-exec/actions/workflows/release.yml)

A mostly GPT-4 created Golang-based tool for executing commands in parallel on Kubernetes pods with specified labels, similar to GNU's `parallel` tool. GPT-4 even helped debug, name the project, create this README (aside from this), create a logo (when DALL-E's not overloaded), and craft the announcement tweet for the project.

## Features

- Execute commands in parallel on multiple Kubernetes pods based on label selectors.
- Specify the target container in the pod.
- Aggregate results from all pods and display them in an easy-to-read format.

## Requirements

- Go 1.16 or higher
- A valid kubeconfig file or in-cluster configuration
- Access to a Kubernetes cluster

## Installation

### Homebrew

You can install `kubectl-parallel-exec` using Homebrew on macOS and Linux:

```bash
brew install cin/kubectl-parallel-exec/kubectl-parallel-exec
```

### From Source

You can also build and install the kubectl-parallel-exec from source. To do this, follow these steps:

1. Ensure that you have Go installed on your system. You can check this by running go version. If you don't have Go installed, you can follow the installation instructions on the [official Go website](https://golang.org/doc/install).

2. Clone the repository:

```sh
git clone https://github.com/your-github-username/kubectl-parallel-exec.git
```

3. Change into the project directory:
```sh
cd kubectl-parallel-exec
```

4. Build the binary:
```sh
go build -o kubectl-parallel-exec
```

5. Move the binary to a directory in your PATH (optional):
```sh
sudo mv kubectl-parallel-exec /usr/local/bin/
```

## Usage
```sh
kubectl-parallel-exec -kubeconfig /path/to/kubeconfig -c container-name -l label-selector -n namespace command-to-execute
```
- `-kubeconfig`: Path to the kubeconfig file. If not provided, KUBECONFIG will be used if defined and then in-cluster configuration will be used.
- `-c`: Container to execute the command against.
- `-l`: Label selector to filter the pods.
- `-n`: Namespace filter
- `-v`: Print the version
- `command-to-execute`: The command to run inside the specified container in each pod.

Example
```sh
kubectl-parallel-exec -kubeconfig ~/.kube/config -c cassandra -l app=cassandra nodetool status
```
This command would execute `nodetool status` on all the Cassandra containers in pods with the label "app=cassandra" in parallel, and then aggregate and display the results.

If authentication is enabled and you don't want to expose credentials, you may have to do things like the following:

```bash
export KPE_OPTS="-l cassandra-cluster-component=cassandra,cassandra-cluster-instance=test-cluster"
alias kpe="k parallel exec $KPE_OPTS"

function knt() {
  local nodetool_command="${*}"
  local nodetool_auth_opts="--ssl -u \$(cat /etc/cassandra-auth-config/admin-role) -pw \$(cat /etc/cassandra-auth-config/admin-password)"
  kpe -c cassandra -- bash -c "nodetool ${nodetool_auth_opts} ${nodetool_command}"
}
```

With this setup, modify `KPE_OPTS` depending on the target pod set; `KUBECONFIG` determines which cluster is being operated against.

## Possible Features

Here are some additional features that could make kubectl-parallel-exec more useful:

Configurable concurrency: Allow users to set the maximum number of concurrent commands that can be executed at once. This would help manage resource usage and prevent overloading the cluster.

Interactive mode: Implement an interactive mode where users can execute multiple commands on selected pods without having to re-run the tool for each command.

Output formatting: Provide options for customizing the output format, such as JSON, YAML, or CSV. This would make it easier to integrate the tool with other systems or to process the output programmatically.

Error handling and retries: Improve error handling by allowing users to specify retry policies for failed command executions or transient errors.

Namespace support: Allow users to specify the namespace(s) to run the commands in, either through command-line flags or configuration files.

Filtering and sorting: Implement options for filtering and sorting the results based on various criteria, such as execution time, pod name, or custom sorting functions.

Logging and audit: Add comprehensive logging and auditing capabilities to track command executions, errors, and other relevant events.

Integration with popular CI/CD tools: Build plugins or integrations with popular continuous integration and deployment tools, such as Jenkins, GitLab CI, or GitHub Actions, to streamline the process of running commands on multiple pods during the deployment process.

Progress tracking and timeout: Implement progress tracking to show the status of command executions in real-time and allow users to set timeouts for long-running commands.

Save and load command sets: Allow users to save sets of commands and their associated options to configuration files, making it easy to run common sets of commands without having to specify all the options every time.

These features could enhance the usability and flexibility of kubectl-parallel-exec, making it an even more valuable tool for Kubernetes users.

## Credits
The initial implementation for this project were provided by ChatGPT from OpenAI. Other than the idea for the project, ChatGPT created nearly all of the code in this repo (even the README.md).
