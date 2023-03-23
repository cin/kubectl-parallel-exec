# k8s-parallel-exec

A Golang-based tool for executing commands in parallel on Kubernetes pods with specified labels, similar to GNU Parallel.

## Features

- Execute commands in parallel on multiple Kubernetes pods based on label selectors.
- Specify the target container in the pod.
- Aggregate results from all pods and display them in an easy-to-read format.

## Requirements

- Go 1.16 or higher
- A valid kubeconfig file or in-cluster configuration
- Access to a Kubernetes cluster

## Installation

1. Clone the repository:

```sh
git clone https://github.com/your-github-username/k8s-parallel-exec.git
```

2. Change into the project directory:
```sh
cd k8s-parallel-exec
```

3. Build the binary:
```sh
go build -o k8s-parallel-exec
```

4. Move the binary to a directory in your PATH (optional):
```sh
sudo mv k8s-parallel-exec /usr/local/bin/
```

## Usage
```sh
k8s-parallel-exec -kubeconfig /path/to/kubeconfig -c container-name -l label-selector command-to-execute
```
- `-kubeconfig`: Path to the kubeconfig file. If not provided, in-cluster configuration will be used.
- `-c`: Container to execute the command against.
- `-l`: Label selector to filter the pods.
- `command-to-execute`: The command to run inside the specified container in each pod.


Example
```sh
k8s-parallel-exec -kubeconfig ~/.kube/config -c cassandra -l app=cassandra nodetool status
```
This command would execute `nodetool status` on all the Cassandra containers in pods with the label "app=cassandra" in parallel, and then aggregate and display the results.

## Possible Features

Here are some additional features that could make k8s-parallel-exec more useful:

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

These features could enhance the usability and flexibility of k8s-parallel-exec, making it an even more valuable tool for Kubernetes users.

## Credits
The initial implementation for this project were provided by ChatGPT from OpenAI. Other than the idea for the project, ChatGPT created nearly all of the code in this repo (even the README.md). 
