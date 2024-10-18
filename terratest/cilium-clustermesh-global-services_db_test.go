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

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"

	"scripts/lib"
)

//go:embed k8s/common/svc-web-app.yaml
var svcWebAppYAML string

func TestCiliumClusterMeshGlobalServiceDB(t *testing.T) {
	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	webAppImage := "ttl.sh/littlejo-webapp:2h"
	webAppYAML = strings.Replace(webAppYAML, "IMAGE", webAppImage, 1)

	for db_index := range contexts {
		t.Run("TestCiliumClusterMeshGlobalServiceDB_"+contexts[db_index], func(t *testing.T) {
			t.Parallel()
			namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

			for i, c := range contexts {
				cm := lib.CreateConfigMapString(clusterNumber, c)
				lib.CreateNamespace(t, c, namespaceName)

				webSvcYAML := svcWebAppYAML
				if i == db_index {
					webSvcYAML = webAppYAML
				}
				lib.ApplyResourceToNamespace(t, c, namespaceName, webSvcYAML)
				lib.ApplyResourceToNamespace(t, c, namespaceName, cm)
				defer lib.DeleteNamespace(t, c, namespaceName)
				defer lib.DeleteResourceToNamespace(t, c, namespaceName, webSvcYAML)
			}

			for _, c := range contexts {
				defer lib.DeleteResourceToNamespace(t, c, namespaceName, clientYAML)
				lib.ApplyResourceToNamespace(t, c, namespaceName, clientYAML)
			}

			for _, c := range contexts {
				pod := lib.RetrieveClient(t, c, namespaceName)
				logsList, _ := lib.WaitForPodLogsNew(t, c, namespaceName, pod, 10, clusterNumber, time.Duration(10)*time.Second)
				logsMap := lib.ValidateLogsDB(t, logsList, contexts[db_index])
				lib.CreateFile(fmt.Sprintf("/tmp/client-db-%s-%s.log", contexts[db_index], c), lib.MapToString(logsMap))
			}
		})
	}
}
