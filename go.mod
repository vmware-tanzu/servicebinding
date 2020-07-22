module github.com/vmware-labs/service-bindings

go 1.14

require (
	go.uber.org/zap v1.15.0
	// normalize k8s.io to v0.17.6
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.6
	knative.dev/pkg v0.0.0-20200702222342-ea4d6e985ba0 // pin to branch release-0.16
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
)
