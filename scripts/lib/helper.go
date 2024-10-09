package lib

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/shell"
)

func CreateConfigMapFile(n int, name string) string {
	file := fmt.Sprintf("/tmp/cm-%s.yaml", name)
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

	outputFile, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	t := template.Must(template.New("configMap").Parse(tmpl))
	err = t.Execute(outputFile, data)
	if err != nil {
		panic(err)
	}

	return file
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
