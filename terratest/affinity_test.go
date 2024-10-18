package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"

	"scripts/lib"
	"scripts/manifests"
)

func TestCiliumClusterMeshGlobalServiceAffinity(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	webAppImage := lib.RetrieveWebAppImage(manifests.WebAppImage)
	deploymentWebAppYAML := strings.Replace(manifests.DeploymentWebAppYAML, "IMAGE", webAppImage, 1)

	index := 0

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(c)
		lib.CreateNamespace(t, c, namespaceName)
		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.SvcWebAppAffYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
		defer lib.DeleteNamespace(t, c, namespaceName)
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.SvcWebAppAffYAML)
	}

	options := k8s.NewKubectlOptions(contexts[len(contexts)-1], "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
	}

	//Step 1: Check Local Affinity
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogs(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
		lib.ValidateLogsDB(t, logsList, c)
	}

	lib.DeleteResourceToNamespace(t, contexts[index], namespaceName, deploymentWebAppYAML)
	pod := lib.RetrieveClient(t, contexts[index], namespaceName)
	lib.WaitForPodAllClustersLogs(t, contexts[index], namespaceName, pod, contexts, clusterNumber, time.Duration(10)*time.Second)

	//Step 2: Check Local Affinity with one failure
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogs(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
		if c != contexts[index] {
			lib.ValidateLogsDB(t, logsList, c)
		} else {
			logsMap := lib.ValidateLogsGlobalServices(t, logsList, contexts)
			lib.CreateFile(fmt.Sprintf("/tmp/client-affinity-%s.log", c), lib.MapToString(logsMap))
		}
	}
}
