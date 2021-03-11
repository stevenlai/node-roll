package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/drain"
)

func main() {
	var nodeName string
	flag.StringVar(&nodeName, "name", "", "Name of node to remove")
	flag.Parse()

	if strings.TrimSpace(nodeName) == "" {
		fmt.Fprintf(os.Stderr, "-name is required\n")
		os.Exit(2)
	}

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err.Error()))
		os.Exit(1)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err.Error()))
		os.Exit(1)
	}

	n, err := findNode(cs, nodeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err.Error()))
		os.Exit(1)
	}

	err = cordonAndDrainNode(cs, n)
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err.Error()))
		os.Exit(1)
	}

}

func findNode(cs *kubernetes.Clientset, nodeName string) (*corev1.Node, error) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/hostname=": nodeName}}
	listOptions := metav1.ListOptions{LabelSelector: labels.Set(labelSelector.MatchLabels).String()}
	matchingNodes, err := cs.CoreV1().Nodes().List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	if len(matchingNodes.Items) < 1 {
		return nil, fmt.Errorf("Unable to find node with name %s.", nodeName)
	}
	return &matchingNodes.Items[0], nil
}

func cordonAndDrainNode(cs *kubernetes.Clientset, n *corev1.Node) error {
	dh := &drain.Helper{
		Client:              cs,
		IgnoreAllDaemonSets: true,
		ErrOut:              os.Stdout,
	}
	err := drain.RunCordonOrUncordon(dh, n, true)
	if err != nil {
		return err
	}
	err = drain.RunNodeDrain(dh, n.Name)
	if err != nil {
		return err
	}

	return err
}
