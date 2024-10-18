package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"

	"scripts/lib"
)

//go:embed k8s/global-load-balancing-affinity/svc-web-app.yaml
var svcWebAppAffYAML string

func TestCiliumClusterMeshGlobalServiceAffinity(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	webAppImage := "ttl.sh/littlejo-webapp:2h"
	deploymentWebAppYAML = strings.Replace(deploymentWebAppYAML, "IMAGE", webAppImage, 1)

	index := 0

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(clusterNumber, c)
		lib.CreateNamespace(t, c, namespaceName)
		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
		lib.ApplyResourceToNamespace(t, c, namespaceName, svcWebAppAffYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
		defer lib.DeleteNamespace(t, c, namespaceName)
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, svcWebAppAffYAML)
	}

	options := k8s.NewKubectlOptions(contexts[len(contexts)-1], "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, clientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, clientYAML)
	}

	//Step 1: Check Local Affinity
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogsNew(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
		lib.ValidateLogsDB(t, logsList, c)
	}

	lib.DeleteResourceToNamespace(t, contexts[index], namespaceName, deploymentWebAppYAML)
	pod := lib.RetrieveClient(t, contexts[index], namespaceName)
	lib.WaitForPodAllClustersLogsNew(t, contexts[index], namespaceName, pod, contexts, clusterNumber, time.Duration(10)*time.Second)

	//Step 2: Check Local Affinity with one failure
	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogsNew(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
		if c != contexts[index] {
			lib.ValidateLogsDB(t, logsList, c)
		} else {
			logsMap := lib.ValidateLogsGlobalServices(t, logsList, contexts)
			lib.CreateFile(fmt.Sprintf("/tmp/client-affinity-%s.log", c), lib.MapToString(logsMap))
		}
	}
}
