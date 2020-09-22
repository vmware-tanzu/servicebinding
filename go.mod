module github.com/vmware-labs/service-bindings

go 1.14

require (
	github.com/google/go-cmp v0.5.2
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	k8s.io/api v0.18.7-rc.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.8
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77 // pin to branch release-0.17
)

replace (
	// normalize k8s.io to v0.17.12
	k8s.io/api => k8s.io/api v0.17.12
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.12
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.12
	k8s.io/client-go => k8s.io/client-go v0.17.12
	k8s.io/code-generator => k8s.io/code-generator v0.17.12
)
