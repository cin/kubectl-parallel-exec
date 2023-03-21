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

## Credits
The initial implementation for this project were provided by ChatGPT from OpenAI. Other than the idea for the project, ChatGPT created nearly all of the code in this repo (even the README.md). 
