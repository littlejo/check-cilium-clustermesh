package manifests

import (
	_ "embed"
)

//go:embed global-load-balancing-affinity/svc-web-app.yaml
var SvcWebAppAffYAML string

//go:embed global-database-shared/svc-shared.yaml
var SvcWebAppTPLYAML string

//go:embed common/deployment-web-app.yaml
var DeploymentWebAppYAML string

//go:embed common/web-app.yaml
var WebAppYAML string

//go:embed common/client.yaml
var ClientYAML string

//go:embed common/svc-web-app.yaml
var SvcWebAppYAML string
