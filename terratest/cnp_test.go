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

func TestCiliumClusterMeshGlobalServiceCiliumNetworkPolicy(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	ciliumNamespace := "kube-system"

	webAppImage := lib.RetrieveWebAppImage(manifests.WebAppImage)
	deploymentWebAppYAML := strings.Replace(manifests.DeploymentWebAppYAML, "IMAGE", webAppImage, 1)

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))
	contextsCiliumClusterName := make(map[string]string)

	for _, c := range contexts {
		contextsCiliumClusterName[c] = lib.RetrieveClusterName(t, c, ciliumNamespace)
	}
	t.Logf("Contexts to Cluster Names map: %v", contextsCiliumClusterName)

	for i, c := range contexts {
		cm := lib.CreateConfigMapString(c)

		nextIndex := (i + 1) % len(contexts)
		nextContext := contexts[nextIndex]
		cnp := lib.CreateCiliumNetworkPolicyString(contextsCiliumClusterName[c], contextsCiliumClusterName[nextContext])

		lib.CreateNamespace(t, c, namespaceName)
		lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
		lib.ApplyResourceToNamespace(t, c, namespaceName, cnp)
		lib.ApplyResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.SvcWebAppYAML)
		defer lib.DeleteNamespace(t, c, namespaceName)
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
	}

	options := lib.NewKubectlOptions(contexts[len(contexts)-1], namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

	for _, c := range contexts {
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
	}

	for i, c := range contexts {
		previousContext := contexts[(i-1+len(contexts))%len(contexts)]
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogs(t, c, namespaceName, pod, 2, clusterNumber, time.Duration(10)*time.Second)
		LogsMap := lib.ValidateLogsOnlyOneValue(t, logsList, previousContext)
		lib.CreateFile(fmt.Sprintf("/tmp/client-cnp-%s.log", c), lib.MapToString(LogsMap))
	}
}
