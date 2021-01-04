module github.com/vmware-labs/service-bindings

go 1.15

require (
	github.com/google/go-cmp v0.5.4
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.20.1
	knative.dev/pkg v0.0.0-20201127013335-0d896b5c87b8 // pin to branch release-0.20 (one available, master until then)
)

replace (
	// normalize k8s.io to v0.18.9
	k8s.io/api => k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.9
	k8s.io/client-go => k8s.io/client-go v0.18.9
	k8s.io/code-generator => k8s.io/code-generator v0.18.9
)
