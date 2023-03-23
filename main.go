package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type ColorCode int

const (
	BrightRed ColorCode = iota + 91
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

const (
	divText      = "----------------------------------------------------\n"
	divColor     = BrightYellow
	podNameColor = BrightCyan
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to the kubeconfig file")
	containerFlag := flag.String("c", "", "Container to execute the command against")
	labelFlag := flag.String("l", "", "Label selector to filter pods")
	flag.Parse()

	if *containerFlag == "" {
		fmt.Println("Error: container name must be specified with -c")
		os.Exit(1)
	}

	if *labelFlag == "" {
		fmt.Println("Error: label selector must be specified with -l")
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: command to execute is required")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: *labelFlag,
	})
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	results := make(chan string, len(pods.Items))

	for _, pod := range pods.Items {
		wg.Add(1)
		go func(p v1.Pod) {
			defer wg.Done()
			output, err := execCommand(config, clientset, p, *containerFlag, args)
			if err != nil {
				results <- fmt.Sprintf("Error executing command on pod %s: %v", p.Name, err)
			} else {
				results <- fmt.Sprintf("%sPod %s\n%s%s",
					colorize(divColor, divText),
					colorize(podNameColor, p.Name),
					colorize(divColor, divText),
					output)
			}
		}(pod)
	}

	wg.Wait()
	close(results)

	for result := range results {
		fmt.Println(result)
	}
}

func colorize(colorCode ColorCode, text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", colorCode, text)
}

func execCommand(config *rest.Config, clientset *kubernetes.Clientset, pod v1.Pod, container string, command []string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: bufio.NewWriter(&stdout),
		Stderr: bufio.NewWriter(&stderr),
	})

	if err != nil {
		return "", err
	}

	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("stderr: %s", stderr.String())
	}

	return stdout.String(), nil
}
