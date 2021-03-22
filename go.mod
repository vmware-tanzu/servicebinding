module github.com/vmware-labs/service-bindings

go 1.15

require (
	github.com/google/go-cmp v0.5.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	k8s.io/code-generator v0.19.7
	knative.dev/pkg v0.0.0-20210216013737-584933f8280b // pin to branch release-0.21
)
