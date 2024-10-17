package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"

	"scripts/lib"
)

func TestCiliumClusterMeshGlobalServiceDB(t *testing.T) {
	contexts, err := lib.GetKubeContexts(t)
	if err != nil {
		fmt.Println(err)
		return
	}
	clusterNumber := len(contexts)
	deploymentName := "client"
	containerName := "client"

	for db_index := range contexts {
		t.Run("TestCiliumClusterMeshGlobalServiceDB_"+contexts[db_index], func(t *testing.T) {
			t.Parallel()
			namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

			for i, c := range contexts {
				cm := lib.CreateConfigMapString(clusterNumber, c)
				file_web := "../web-server/k8s/global-database/global-svc.yaml"
				if i == db_index {
					file_web = "../web-server/k8s/common/web-app.yaml"
				}
				webResourcePath, err := filepath.Abs(file_web)
				require.NoError(t, err)

				options := k8s.NewKubectlOptions(c, "", namespaceName)

				k8s.CreateNamespace(t, options, namespaceName)
				defer k8s.DeleteNamespace(t, options, namespaceName)
				defer k8s.KubectlDelete(t, options, webResourcePath)

				k8s.KubectlApplyFromString(t, options, cm)
				k8s.KubectlApply(t, options, webResourcePath)
			}

			for _, c := range contexts {
				clientResourcePath, err := filepath.Abs("../web-server/k8s/common/client.yaml")
				require.NoError(t, err)

				options := k8s.NewKubectlOptions(c, "", namespaceName)

				defer k8s.KubectlDelete(t, options, clientResourcePath)

				k8s.KubectlApply(t, options, clientResourcePath)
			}

			for _, c := range contexts {
				options := k8s.NewKubectlOptions(c, "", namespaceName)
				filters := metav1.ListOptions{
					LabelSelector: "app=client",
				}
				k8s.WaitUntilDeploymentAvailable(t, options, deploymentName, 60, time.Duration(1)*time.Second)
				pod := k8s.ListPods(t, options, filters)[0]
				lib.WaitForPodLogs(t, options, pod.Name, containerName, clusterNumber, time.Duration(10)*time.Second)
				logs := k8s.GetPodLogs(t, options, &pod, containerName)
				logsList := strings.Split(logs, "\n")
				logsMap := lib.Uniq(logsList)
				t.Log("Value of logs is:", lib.MapToString(logsMap))
				lib.CreateFile(fmt.Sprintf("/tmp/client-db-%s-%s.log", contexts[db_index], c), lib.MapToString(logsMap))
				require.Contains(t, logsList, contexts[db_index])
				require.Equal(t, len(logsMap), 1)
			}
		})
	}
}
