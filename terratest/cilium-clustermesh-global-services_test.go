package test

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

func TestCiliumClusterMeshGlobalService(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)

	webAppImage := "ttl.sh/littlejo-webapp:2h"
	webAppYAML := strings.Replace(manifests.WebAppYAML, "IMAGE", webAppImage, 1)

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(c)

		lib.CreateNamespace(t, c, namespaceName)
		defer lib.DeleteNamespace(t, c, namespaceName)
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, webAppYAML)

		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
		lib.ApplyResourceToNamespace(t, c, namespaceName, webAppYAML)
	}

	options := k8s.NewKubectlOptions(contexts[len(contexts)-1], "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
	}

	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		lib.WaitForPodAllClustersLogs(t, c, namespaceName, pod, contexts, clusterNumber, time.Duration(10)*time.Second)
		logsList := lib.GetLogsList(t, c, namespaceName, pod)
		logsMap := lib.ValidateLogsGlobalServices(t, logsList, contexts)
		lib.CreateFile(fmt.Sprintf("/tmp/client-%s.log", c), lib.MapToString(logsMap))
	}
}
