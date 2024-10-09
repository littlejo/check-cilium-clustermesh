package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
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

func TestCiliumClusterMeshGlobalService(t *testing.T) {
	t.Parallel()

	contexts, err := GetKubeContexts(t)
	if err != nil {
		fmt.Println(err)
		return
	}
	cluster_number := len(contexts)
	podName := "script-runner"
	containerName := "script-container"

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := CreateConfigMapFile(cluster_number, c)
		cmResourcePath, err := filepath.Abs(cm)
		webResourcePath, err := filepath.Abs("../web-server/k8s/pod-svc.yaml")
		require.NoError(t, err)

		options := k8s.NewKubectlOptions(c, "", namespaceName)

		k8s.CreateNamespace(t, options, namespaceName)
		defer k8s.DeleteNamespace(t, options, namespaceName)
		defer k8s.KubectlDelete(t, options, webResourcePath)

		k8s.KubectlApply(t, options, cmResourcePath)
		k8s.KubectlApply(t, options, webResourcePath)
	}

	for _, c := range contexts {
		options := k8s.NewKubectlOptions(c, "", namespaceName)
		k8s.WaitUntilPodAvailable(t, options, "go-web-server-pod", 10, time.Duration(2)*time.Second)
	}

	for _, c := range contexts {
		clientResourcePath, err := filepath.Abs("../web-server/k8s/pod-script.yaml")
		require.NoError(t, err)

		options := k8s.NewKubectlOptions(c, "", namespaceName)

		defer k8s.KubectlDelete(t, options, clientResourcePath)

		k8s.KubectlApply(t, options, clientResourcePath)
	}

	for _, c := range contexts {
		options := k8s.NewKubectlOptions(c, "", namespaceName)
		pods := k8s.GetPod(t, options, podName)
		k8s.WaitUntilPodAvailable(t, options, podName, 10, time.Duration(10)*time.Second)
		WaitForPodLogs(t, options, podName, containerName, cluster_number, time.Duration(10)*time.Second)
		logs := k8s.GetPodLogs(t, options, pods, containerName)
		t.Log("Value of logs is:", logs)
		CreateFile(fmt.Sprintf("/tmp/runner-%s.log", c), logs)
		numberOfLines := strings.Count(logs, "\n") + 1
		require.Equal(t, numberOfLines, cluster_number)
	}
}
