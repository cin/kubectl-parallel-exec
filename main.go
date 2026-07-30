package main

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	brightYellow = 93
	brightCyan   = 96

	divText      = "----------------------------------------------------\n"
	divColor     = brightYellow
	podNameColor = brightCyan

	defaultConcurrency = 16
)

var version = "dev"

type PodResult struct {
	podName string
	output  string
	err     error
	elapsed time.Duration
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to the kubeconfig file")
	container := flag.String("c", "", "Container to execute the command against")
	labelSelector := flag.String("l", "", "Label selector to filter pods")
	namespace := flag.String("n", "", "Namespace filter")
	concurrency := flag.Int("j", defaultConcurrency, "Maximum concurrent execs (0 = unlimited)")
	timeout := flag.Duration("timeout", 0, "Per-pod exec timeout (0 = no timeout)")
	versionFlag := flag.Bool("v", false, "Print the version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	if *container == "" {
		fatal("container name must be specified with -c")
	}

	if *labelSelector == "" {
		fatal("label selector must be specified with -l")
	}

	if *concurrency < 0 {
		fatal("concurrency (-j) must be >= 0")
	}

	args := flag.Args()
	if len(args) == 0 {
		fatal("command to execute is required")
	}

	kubeconfigPath := selectKubeconfig(*kubeconfig, os.Getenv("KUBECONFIG"))

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		kubeconfigErr := err
		config, err = rest.InClusterConfig()
		if err != nil {
			fatal("failed to load configuration: %v", errors.Join(kubeconfigErr, err))
		}
	}

	tuneClientThroughput(config, *concurrency)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fatal("failed to create kubernetes client: %v", err)
	}

	pods, err := clientset.CoreV1().Pods(*namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: *labelSelector,
	})
	if err != nil {
		fatal("failed to list pods: %v", err)
	}

	results := runParallelExec(config, clientset, pods.Items, *container, args, *concurrency, *timeout)
	sortPodResults(results)

	failed := false
	for _, result := range results {
		fmt.Print(formatPodResult(result))
		if result.err != nil {
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func selectKubeconfig(flagValue, envValue string) string {
	if flagValue != "" {
		return flagValue
	}

	return envValue
}

func tuneClientThroughput(config *rest.Config, concurrency int) {
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	config.QPS = float32(concurrency)
	config.Burst = concurrency * 2
}

func runParallelExec(
	config *rest.Config,
	clientset *kubernetes.Clientset,
	pods []v1.Pod,
	container string,
	command []string,
	concurrency int,
	timeout time.Duration,
) []PodResult {
	if len(pods) == 0 {
		return nil
	}

	limit := concurrency
	if limit <= 0 {
		limit = len(pods)
	}

	sem := make(chan struct{}, limit)
	resultsChan := make(chan PodResult, len(pods))
	var wg sync.WaitGroup

	for _, pod := range pods {
		wg.Add(1)
		go func(name, ns string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resultsChan <- execCommand(config, clientset, name, ns, container, command, timeout)
		}(pod.Name, pod.Namespace)
	}

	wg.Wait()
	close(resultsChan)

	results := make([]PodResult, 0, len(pods))
	for result := range resultsChan {
		results = append(results, result)
	}
	return results
}

func sortPodResults(results []PodResult) {
	slices.SortFunc(results, func(a, b PodResult) int {
		return cmp.Compare(a.podName, b.podName)
	})
}

func colorize(colorCode int, text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", colorCode, text)
}

func formatPodResult(result PodResult) string {
	header := fmt.Sprintf("%sPod %s - %s\n%s",
		colorize(divColor, divText),
		colorize(podNameColor, result.podName),
		result.elapsed.String(),
		colorize(divColor, divText))

	output := result.output
	if output != "" && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	if result.err != nil {
		return fmt.Sprintf("%sError executing command: %v\n%s", header, result.err, output)
	}

	return header + output
}

// combineOutput concatenates stdout then stderr; unlike kubectl, the two
// streams are not interleaved in arrival order.
func combineOutput(stdout, stderr string) string {
	return stdout + stderr
}

func execCommand(
	config *rest.Config,
	clientset *kubernetes.Clientset,
	podName, namespace, container string,
	command []string,
	timeout time.Duration,
) PodResult {
	start := time.Now()
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	reqURL := req.URL()
	exec, err := newExecutor(config, reqURL)
	if err != nil {
		return PodResult{podName, "", err, time.Since(start)}
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	output := combineOutput(stdout.String(), stderr.String())
	if err != nil {
		return PodResult{podName, output, err, time.Since(start)}
	}

	return PodResult{podName, output, nil, time.Since(start)}
}

func newExecutor(config *rest.Config, reqURL *url.URL) (remotecommand.Executor, error) {
	spdyExec, err := remotecommand.NewSPDYExecutor(config, "POST", reqURL)
	if err != nil {
		return nil, err
	}

	// WebSocketExecutor must use GET per RFC 6455 Sec. 4.1.
	websocketExec, err := remotecommand.NewWebSocketExecutor(config, "GET", reqURL.String())
	if err != nil {
		return nil, err
	}

	return remotecommand.NewFallbackExecutor(websocketExec, spdyExec, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}
