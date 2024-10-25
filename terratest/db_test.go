package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"

	"scripts/lib"
	"scripts/manifests"
)

func TestCiliumClusterMeshGlobalServiceDB(t *testing.T) {
	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	webAppImage := lib.RetrieveWebAppImage(manifests.WebAppImage)
	deploymentWebAppYAML := strings.Replace(manifests.DeploymentWebAppYAML, "IMAGE", webAppImage, 1)

	contextsTest := contexts[:2]

	for db_index := range contextsTest {
		t.Run("TestCiliumClusterMeshGlobalServiceDB_"+contexts[db_index], func(t *testing.T) {
			t.Parallel()
			namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

			for i, c := range contexts {
				cm := lib.CreateConfigMapString(c)
				lib.CreateNamespace(t, c, namespaceName)

				webSvcYAML := manifests.SvcWebAppYAML
				if i == db_index {
					lib.ApplyResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
					defer lib.DeleteResourceToNamespace(t, c, namespaceName, deploymentWebAppYAML)
				}
				lib.ApplyResourceToNamespace(t, c, namespaceName, webSvcYAML)
				lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
				defer lib.DeleteNamespace(t, c, namespaceName)
			}

			for _, c := range contexts {
				defer lib.DeleteResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
				lib.ApplyResourceToNamespace(t, c, namespaceName, manifests.ClientYAML)
			}

			for _, c := range contexts {
				pod := lib.RetrieveClient(t, c, namespaceName)
				logsList, _ := lib.WaitForPodLogs(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
				logsMap := lib.ValidateLogsOnlyOneValue(t, logsList, contexts[db_index])
				lib.CreateFile(fmt.Sprintf("/tmp/client-db-%s-%s.log", contexts[db_index], c), lib.MapToString(logsMap))
			}
		})
	}
}
