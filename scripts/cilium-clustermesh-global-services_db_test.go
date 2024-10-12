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
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	if err != nil {
		fmt.Println(err)
		return
	}
	cluster_number := len(contexts)
	deploymentName := "client"
	containerName := "client"

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for i, c := range contexts {
		cm := lib.CreateConfigMapFile(cluster_number, c)
		cmResourcePath, err := filepath.Abs(cm)
		file_web := "../web-server/k8s/global-database/global-svc.yaml"
		if i == 0 {
			file_web = "../web-server/k8s/common/web-app.yaml"
		}
		webResourcePath, err := filepath.Abs(file_web)
		require.NoError(t, err)

		options := k8s.NewKubectlOptions(c, "", namespaceName)

		k8s.CreateNamespace(t, options, namespaceName)
		defer k8s.DeleteNamespace(t, options, namespaceName)
		defer k8s.KubectlDelete(t, options, webResourcePath)

		k8s.KubectlApply(t, options, cmResourcePath)
		k8s.KubectlApply(t, options, webResourcePath)
	}

	for _, c := range contexts {
		clientResourcePath, err := filepath.Abs("../web-server/k8s/global-database/client.yaml")
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
		p := k8s.GetPod(t, options, pod.Name)
		lib.WaitForPodLogs(t, options, pod.Name, containerName, cluster_number, time.Duration(10)*time.Second)
		logs := k8s.GetPodLogs(t, options, p, containerName)
		t.Log("Value of logs is:", logs)
		lib.CreateFile(fmt.Sprintf("/tmp/client-db-%s.log", c), logs)
		numberOfLines := strings.Count(logs, "\n") + 1
		require.Equal(t, numberOfLines, 1)
		require.Contains(t, logs, "100")
		require.Contains(t, logs, contexts[0])
	}
}
