package lib

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/shell"
)

func CreateConfigMapString(n int, name string) string {
	tmpl := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster
data:
  CLUSTER: "{{.Cluster}}"
  CLUSTERS_NUMBER: "{{.ClustersNumber}}"`

	data := struct {
		Cluster        string
		ClustersNumber int
	}{
		Cluster:        name,
		ClustersNumber: n,
	}

	var result bytes.Buffer
	t := template.Must(template.New("configMap").Parse(tmpl))
	err := t.Execute(&result, data)
	if err != nil {
		panic(err)
	}

	return result.String()
}

func CreateCiliumNetworkPolicyString(endpointClusterName string, ingressClusterName string) string {
	tmpl := `apiVersion: "cilium.io/v2"
kind: CiliumNetworkPolicy
metadata:
  name: "allow-client-access-to-web-app"
spec:
  description: "Allow traffic from clients in {{.IngressClusterName}} to access the web application in {{.EndpointClusterName}}"
  endpointSelector:
    matchLabels:
      app: web-app
      io.cilium.k8s.policy.cluster: {{.EndpointClusterName}}
  ingress:
  - fromEndpoints:
    - matchLabels:
        app: client
        io.cilium.k8s.policy.cluster: {{.IngressClusterName}}`

	data := struct {
		EndpointClusterName string
		IngressClusterName  string
	}{
		EndpointClusterName: endpointClusterName,
		IngressClusterName:  ingressClusterName,
	}

	var result bytes.Buffer
	t := template.Must(template.New("ciliumNetworkPolicy").Parse(tmpl))
	err := t.Execute(&result, data)
	if err != nil {
		panic(err)
	}

	return result.String()
}

func CreateFile(fileName string, content string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}

	return nil
}

func WaitForPodLogs(t *testing.T, options *k8s.KubectlOptions, podName string, containerName string, maxRetries int, retryInterval time.Duration) (string, error) {
	var logs string
	pods := k8s.GetPod(t, options, podName)

	for i := 0; i < maxRetries; i++ {
		logs = k8s.GetPodLogs(t, options, pods, containerName)

		if logs != "" {
			return logs, nil
		}

		time.Sleep(retryInterval)
	}

	return "", fmt.Errorf("Impossible to retrieve after %d tries", maxRetries)
}

func WaitForPodAllClustersLogs(t *testing.T, options *k8s.KubectlOptions, podName string, containerName string, contexts []string, maxRetries int, retryInterval time.Duration) (string, error) {
	var logs string
	pods := k8s.GetPod(t, options, podName)

	for i := 0; i < maxRetries; i++ {
		logs = k8s.GetPodLogs(t, options, pods, containerName)
		logsList := strings.Split(logs, "\n")
		allPresent := true
		for _, c := range contexts {
			if !contains(logsList, c) {
				allPresent = false
			}
		}

		if allPresent {
			return logs, nil
		}

		time.Sleep(retryInterval)
	}

	return "", fmt.Errorf("Impossible to retrieve after %d tries", maxRetries)
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func GetKubeContexts(t *testing.T) ([]string, error) {
	kubectlCmd := shell.Command{
		Command: "kubectl",
		Args:    []string{"config", "get-contexts", "-o", "name"},
	}

	output, err := shell.RunCommandAndGetStdOutE(t, kubectlCmd)
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	contexts := strings.Split(strings.TrimSpace(output), "\n")
	return contexts, nil
}

func Uniq(list []string) map[string]int {
	occurrences := make(map[string]int)

	for _, v := range list {
		occurrences[v]++
	}

	return occurrences
}

func MapToString(m map[string]int) string {
	var sb strings.Builder
	sb.WriteString("{") // Commence avec une accolade ouvrante

	first := true
	for key, value := range m {
		if !first {
			sb.WriteString(", ") // Ajoute une virgule entre les paires clé-valeur
		}
		first = false
		sb.WriteString(fmt.Sprintf("%s: %d", key, value)) // Ajoute "clé: valeur" au builder
	}

	sb.WriteString("}") // Termine avec une accolade fermante
	return sb.String()
}
