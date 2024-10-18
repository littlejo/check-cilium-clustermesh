package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"

	"scripts/lib"
	"scripts/manifests"
)

func TestCiliumClusterMeshGlobalServiceShared(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	blue := contexts[0]
	green := contexts[clusterNumber-1]
	blueGreenContexts := []string{blue, green}

	webAppImage := lib.RetrieveWebAppImage(manifests.WebAppImage)
	deploymentWebAppYAML := strings.Replace(manifests.DeploymentWebAppYAML, "IMAGE", webAppImage, 1)

	sharedSvcWebAppYAML := strings.Replace(manifests.SvcWebAppTPLYAML, "SHARED", "true", 1)
	unsharedSvcWebAppYAML := strings.Replace(manifests.SvcWebAppTPLYAML, "SHARED", "false", 1)

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(c)

		lib.CreateNamespace(t, c, namespaceName)
		defer lib.DeleteNamespace(t, c, namespaceName)
		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)

		webSvcYAML := unsharedSvcWebAppYAML
		if c == blue {
			webSvcYAML = sharedSvcWebAppYAML
		}
		lib.ApplyResourceToNamespace(t, c, namespaceName, webSvcYAML)
		if c == blue || c == green {
			lib.ApplyResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
			defer lib.DeleteResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
		}
	}

	options := k8s.NewKubectlOptions(green, "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
	}

	//Step Blue: Check
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		//TOFIX
		logsList, err := lib.WaitForPodLogs(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
		require.NoError(t, err, "Error waiting for pod logs in context: %s", c)

		var logsMap map[string]int
		if c != green {
			logsMap = lib.ValidateLogsOnlyOneValue(t, logsList, blue)
		} else {
			logsMap = lib.ValidateLogsAllValues(t, logsList, blueGreenContexts)
		}
		lib.CreateFile(fmt.Sprintf("/tmp/client-shared-blue-%s.log", c), lib.MapToString(logsMap))
	}

	lib.ApplyResourceToNamespace(t, blue, namespaceName, unsharedSvcWebAppYAML)
	lib.ApplyResourceToNamespace(t, green, namespaceName, sharedSvcWebAppYAML)

	indexes := make(map[string]int)
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		indexes[c] = len(lib.GetLogsList(t, c, namespaceName, pod))
	}

	//Step Green: Check
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, err := lib.WaitForPodAllClustersLogs(t, c, namespaceName, pod, blueGreenContexts, clusterNumber, time.Duration(10)*time.Second)
		require.NoError(t, err, "Error waiting for pod logs in context: %s", c)
		endLogsList := logsList[indexes[c]:]
		var logsMap map[string]int
		if c != blue {
			logsMap = lib.ValidateLogsOnlyOneValue(t, endLogsList, green)
		} else {
			logsMap = lib.ValidateLogsAllValues(t, endLogsList, blueGreenContexts)
		}
		lib.CreateFile(fmt.Sprintf("/tmp/client-shared-green-%s.log", c), lib.MapToString(logsMap))
	}
}
