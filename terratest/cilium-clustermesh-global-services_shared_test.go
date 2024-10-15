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

func TestCiliumClusterMeshGlobalServiceShared(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	if err != nil {
		fmt.Println(err)
		return
	}
	blue := contexts[0]
	green := contexts[len(contexts)-1]
	clusterNumber := len(contexts)
	deploymentName := "client"
	containerName := "client"

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for i, c := range contexts {
		cm := lib.CreateConfigMapString(clusterNumber, c)
		webPath := "../web-server/k8s/global-database-shared/web-app.yaml"
		if i == 0 {
			webPath = "../web-server/k8s/global-database-shared/web-app-shared.yaml"
		} else if i == len(contexts)-1 {
			webPath = "../web-server/k8s/global-database-shared/web-app-shared-false.yaml"
		}

		webResourcePath, err := filepath.Abs(webPath)
		require.NoError(t, err)

		options := k8s.NewKubectlOptions(c, "", namespaceName)

		k8s.CreateNamespace(t, options, namespaceName)
		defer k8s.DeleteNamespace(t, options, namespaceName)
		defer k8s.KubectlDelete(t, options, webResourcePath)

		k8s.KubectlApplyFromString(t, options, cm)
		k8s.KubectlApply(t, options, webResourcePath)
	}

	options := k8s.NewKubectlOptions(green, "", namespaceName)
	k8s.WaitUntilDeploymentAvailable(t, options, "web-app", 60, time.Duration(1)*time.Second)

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
		t.Log("Value of logs is:", logs)
		logsList := strings.Split(logs, "\n")
		contextsAnalyze := []string{blue}
		if c == green {
			contextsAnalyze = []string{blue, green}
		}
		for _, c := range contextsAnalyze {
			require.Contains(t, logsList, c)
		}
	}

	options = k8s.NewKubectlOptions(blue, "", namespaceName)
	webSwitchPath := "../web-server/k8s/global-database-shared/web-app-shared-false.yaml"
	webResourceSwitchPath, err := filepath.Abs(webSwitchPath)
	k8s.KubectlApply(t, options, webResourceSwitchPath)

	options = k8s.NewKubectlOptions(green, "", namespaceName)
	webSwitchPath = "../web-server/k8s/global-database-shared/web-app-shared.yaml"
	webResourceSwitchPath, err = filepath.Abs(webSwitchPath)
	k8s.KubectlApply(t, options, webResourceSwitchPath)

	waitContexts := []string{blue, green}

	filters := metav1.ListOptions{
		LabelSelector: "app=client",
	}

	for _, c := range contexts {
		options = k8s.NewKubectlOptions(c, "", namespaceName)
		pod := k8s.ListPods(t, options, filters)[0]
		lib.WaitForPodAllClustersLogs(t, options, pod.Name, containerName, waitContexts, clusterNumber, time.Duration(10)*time.Second)
	}

	for _, c := range contexts {
		options := k8s.NewKubectlOptions(c, "", namespaceName)
		filters := metav1.ListOptions{
			LabelSelector: "app=client",
		}
		pod := k8s.ListPods(t, options, filters)[0]
		logs := k8s.GetPodLogs(t, options, &pod, containerName)
		logsList := strings.Split(logs, "\n")
		contextsAnalyze := []string{blue, green}
		lib.CreateFile(fmt.Sprintf("/tmp/client-shared-%s.log", c), logs)
		for _, c := range contextsAnalyze {
			require.Contains(t, logsList, c)
		}
		t.Log("Value of pod name is:", pod.Name)
		t.Log("Value of logs is:", logs)
	}
}
