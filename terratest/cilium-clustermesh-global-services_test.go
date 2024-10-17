package test

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"

	"scripts/lib"
)

//go:embed k8s/common/web-app.yaml
var webAppYAML string

//go:embed k8s/common/client.yaml
var clientYAML string

func TestCiliumClusterMeshGlobalService(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	deploymentName := "client"
	containerName := "client"

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(clusterNumber, c)

		lib.CreateNamespace(t, c, namespaceName)
		defer lib.DeleteNamespace(t, c, namespaceName)
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, webAppYAML)

		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
		lib.ApplyResourceToNamespace(t, c, namespaceName, webAppYAML)
	}

	options := k8s.NewKubectlOptions(contexts[len(contexts)-1], "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, clientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, clientYAML)
	}

	for _, c := range contexts {
		options := k8s.NewKubectlOptions(c, "", namespaceName)
		filters := metav1.ListOptions{
			LabelSelector: "app=client",
		}
		k8s.WaitUntilDeploymentAvailable(t, options, deploymentName, 60, time.Duration(1)*time.Second)
		pod := k8s.ListPods(t, options, filters)[0]
		lib.WaitForPodAllClustersLogs(t, options, pod.Name, containerName, contexts, clusterNumber, time.Duration(10)*time.Second)
		logs := k8s.GetPodLogs(t, options, &pod, containerName)
		logsList := strings.Split(logs, "\n")
		LogsMap := lib.Uniq(logsList)
		t.Log("Value of pod name is:", pod.Name)
		t.Log("Value of logs is:", lib.MapToString(LogsMap))
		lib.CreateFile(fmt.Sprintf("/tmp/client-%s.log", c), lib.MapToString(LogsMap))
		require.Equal(t, len(LogsMap), clusterNumber)
		for _, c := range contexts {
			require.Contains(t, logsList, c)
		}
	}
}
