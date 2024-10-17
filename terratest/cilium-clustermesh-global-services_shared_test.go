package test

//go test -v cilium-clustermesh-global-services_test.go
//go test -c -o test-binary cilium-clustermesh-global-services_test.go
//./test-binary -test.v

import (
	_ "embed"
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

//go:embed k8s/global-database-shared/svc-shared.yaml
var svcWebAppYAML string

//go:embed k8s/common/deployment-web-app.yaml
var deploymentWebAppYAML string

func TestCiliumClusterMeshGlobalServiceShared(t *testing.T) {
	t.Parallel()

	contexts, err := lib.GetKubeContexts(t)
	require.NoError(t, err, "Failed to get Kube contexts")
	clusterNumber := len(contexts)
	blue := contexts[0]
	green := contexts[clusterNumber-1]

	webAppImage := "ttl.sh/littlejo-webapp:2h"
	deploymentWebAppYAML = strings.Replace(deploymentWebAppYAML, "IMAGE", webAppImage, 1)

	sharedSvcWebAppYAML := strings.Replace(svcWebAppYAML, "SHARED", "true", 1)
	unsharedSvcWebAppYAML := strings.Replace(svcWebAppYAML, "SHARED", "false", 1)

	namespaceName := fmt.Sprintf("cilium-cmesh-test-%s", strings.ToLower(random.UniqueId()))

	for _, c := range contexts {
		cm := lib.CreateConfigMapString(clusterNumber, c)

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
		defer lib.DeleteResourceToNamespace(t, c, namespaceName, clientYAML)
		lib.ApplyResourceToNamespace(t, c, namespaceName, clientYAML)
	}

	for _, c := range contexts {
		pod := lib.RetrieveClient(t, c, namespaceName)
		logsList, _ := lib.WaitForPodLogsNew(t, c, namespaceName, pod, clusterNumber, time.Duration(10)*time.Second)
		logsMap := lib.ValidateLogsSharedStep1(t, logsList, c, []string{blue, green})
		lib.CreateFile(fmt.Sprintf("/tmp/client-shared-blue-%s.log", c), lib.MapToString(logsMap))
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
		lib.WaitForPodAllClustersLogs(t, options, pod.Name, "", waitContexts, clusterNumber, time.Duration(10)*time.Second)
	}

	for _, c := range contexts {
		options := k8s.NewKubectlOptions(c, "", namespaceName)
		filters := metav1.ListOptions{
			LabelSelector: "app=client",
		}
		pod := k8s.ListPods(t, options, filters)[0]
		logs := k8s.GetPodLogs(t, options, &pod, "")
		logsList := strings.Split(logs, "\n")
		LogsMap := lib.Uniq(logsList)
		contextsAnalyze := []string{blue, green}
		lib.CreateFile(fmt.Sprintf("/tmp/client-shared-green-%s.log", c), lib.MapToString(LogsMap))
		for _, c := range contextsAnalyze {
			require.Contains(t, logsList, c)
		}
		require.Equal(t, len(LogsMap), 2)
	}
}
