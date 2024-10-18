package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateConfigMapString(name string) string {
	tmpl := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster
data:
  CLUSTER: "{{.Cluster}}"`

	data := struct {
		Cluster string
	}{
		Cluster: name,
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
	jsonData, _ := json.Marshal(m)
	return string(jsonData)
}

func ApplyResourceToNamespace(t *testing.T, context string, namespaceName string, manifest string) {
	resourcesMap, err := ParseYAMLResources(manifest)
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}
	for resource, name := range resourcesMap {
		Log(t, context, "✨ Creating "+resource+" "+name)
	}
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	k8s.KubectlApplyFromString(t, options, manifest)
}

func DeleteResourceToNamespace(t *testing.T, context string, namespaceName string, manifest string) {
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	k8s.KubectlDeleteFromString(t, options, manifest)
}

func CreateNamespace(t *testing.T, context string, namespaceName string) {
	Log(t, context, "✨ Creating namespace "+namespaceName)
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	k8s.CreateNamespace(t, options, namespaceName)
}

func DeleteNamespace(t *testing.T, context string, namespaceName string) {
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	k8s.DeleteNamespace(t, options, namespaceName)
}

func RetrieveClient(t *testing.T, context string, namespaceName string) corev1.Pod {
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	filters := metav1.ListOptions{
		LabelSelector: "app=client",
	}
	k8s.WaitUntilDeploymentAvailable(t, options, "client", 60, time.Duration(1)*time.Second)
	return k8s.ListPods(t, options, filters)[0]
}

func RetrieveClusterName(t *testing.T, context string, namespaceName string) string {
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	ciliumConfigMap := k8s.GetConfigMap(t, options, "cilium-config")
	clusterName, exists := ciliumConfigMap.Data["cluster-name"]
	assert.True(t, exists, "Key 'cluster-name' should exist in the ConfigMap")
	return clusterName
}

func RetrieveWebAppImage(webAppImage string) string {
	webAppImageEnv := os.Getenv("WEBAPP_IMAGE")
	if webAppImageEnv != "" {
		return webAppImageEnv
	}
	return webAppImage
}

func GetLogsList(t *testing.T, context string, namespaceName string, pod corev1.Pod) []string {
	options := k8s.NewKubectlOptions(context, "", namespaceName)
	logs := k8s.GetPodLogs(t, options, &pod, "")
	return strings.Split(logs, "\n")
}

func WaitForPodLogs(t *testing.T, context string, namespaceName string, pod corev1.Pod, lineNumber int, maxRetries int, retryInterval time.Duration) ([]string, error) {
	var logsList []string
	for i := 0; i < maxRetries; i++ {
		logsList = GetLogsList(t, context, namespaceName, pod)

		if len(logsList) > lineNumber {
			return logsList, nil
		}

		time.Sleep(retryInterval)
	}

	return logsList, fmt.Errorf("Impossible to retrieve after %d tries", maxRetries)
}

func WaitForPodAllClustersLogs(t *testing.T, context string, namespaceName string, pod corev1.Pod, contexts []string, maxRetries int, retryInterval time.Duration) ([]string, error) {
	var logsList []string
	for i := 0; i < maxRetries; i++ {
		logsList = GetLogsList(t, context, namespaceName, pod)
		allPresent := true
		for _, c := range contexts {
			if !contains(logsList, c) {
				allPresent = false
			}
		}

		if allPresent {
			return logsList, nil
		}

		time.Sleep(retryInterval)
	}

	return logsList, fmt.Errorf("Impossible to retrieve after %d tries", maxRetries)
}

func Log(t *testing.T, context string, message string) {
	t.Logf("[%s] [%s] %s", t.Name(), context, message)
}
