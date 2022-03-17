
# Service Bindings for Kubernetes

![CI](https://github.com/vmware-tanzu/servicebinding/workflows/CI/badge.svg?branch=main)
[![GoDoc](https://godoc.org/github.com/vmware-tanzu/servicebinding?status.svg)](https://godoc.org/github.com/vmware-tanzu/servicebinding)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmware-tanzu/servicebinding)](https://goreportcard.com/report/github.com/vmware-tanzu/servicebinding)
[![codecov](https://codecov.io/gh/vmware-tanzu/servicebinding/branch/main/graph/badge.svg)](https://codecov.io/gh/vmware-tanzu/servicebinding)


Service Bindings for Kubernetes implements the [Service Binding Specification for Kubernetes](https://servicebinding.io/) v1.0.

This implementation provides support for:
- [Provisioned Service](https://github.com/servicebinding/spec/tree/v1.0.0#provisioned-service)
- [Workload Projection](https://github.com/servicebinding/spec/tree/v1.0.0#workload-projection)
- [Service Binding](https://github.com/servicebinding/spec/tree/v1.0.0#service-binding)
- [Direct Secret Reference](https://github.com/servicebinding/spec/tree/v1.0.0#direct-secret-reference)
- [Role-Based Access Control (RBAC)](https://github.com/servicebinding/spec/tree/v1.0.0#role-based-access-control-rbac)

The following are not implemented:
- [Workload Resource Mapping](https://github.com/servicebinding/spec/tree/v1.0.0#workload-resource-mapping)
- Extensions including:
  - [Binding Secret Generation Strategies](https://github.com/servicebinding/spec/blob/v1.0.0/extensions/secret-generation.md)

Equivalent capabilities from the v1.0.0-rc3 (servicebinding.io/v1alpha3) version of the spec are also supported. There are no significant API or runtime changes between v1alpha3 and v1beta1 versions.

## Try it out

Prerequisites:
- a Kubernetes 1.18+ cluster

Using the [latest release](https://github.com/vmware-tanzu/servicebinding/releases/latest) is recommended.

### Build from source

We use [Golang](https://golang.org) and [`ko`](https://github.com/google/ko) to build the CRD and reconciler, and [`kapp`](https://get-kapp.io) to deploy them.

From within the cloned directory for this project, run:

```
kapp deploy -a service-bindings -f <(ko resolve -f config)
```

#### Uninstall

```
kapp delete -a service-bindings
```

## Collecting logs from service binding manager

Retrieve pod logs from the `manager` running in the `service-bindings` namespace.

  ```bash
  kubectl -n service-bindings logs -l role=manager
  ```

For example:

  ```bash
  2021/11/05 15:25:28 Registering 3 clients
  2021/11/05 15:25:28 Registering 3 informer factories
  2021/11/05 15:25:28 Registering 7 informers
  2021/11/05 15:25:28 Registering 8 controllers
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.483823208Z","caller":"logging/config.go:116","message":"Successfully created the logger."}
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.48392361Z","caller":"logging/config.go:117","message":"Logging level set to: info"}
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.483999911Z","caller":"logging/config.go:79","message":"Fetch GitHub commit ID from kodata failed","error":"open /var/run/ko/HEAD: no such file or directory"}
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.484035711Z","logger":"webhook","caller":"profiling/server.go:64","message":"Profiling enabled: false"}
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.522884909Z","logger":"webhook","caller":"leaderelection/context.go:46","message":"Running with Standard leader election"}
  {"severity":"INFO","timestamp":"2021-11-05T15:25:28.523358615Z","logger":"webhook","caller":"provisionedservice/controller.go:31","message":"Setting up event handlers."}
  ...
    {"severity":"ERROR","timestamp":"2021-11-17T15:00:24.561881861Z","logger":"webhook","caller":"controller/controller.go:548","message":"Reconcile error","duration":"167.902µs","error":"deployments.apps \"spring-petclinic\" not found","stacktrace":"knative.dev/pkg/controller.(*Impl).handleErr\n\tknative.dev/pkg@v0.0.0-20210331065221-952fdd90dbb0/controller/controller.go:548\nknative.dev/pkg/controller.(*Impl).processNextWorkItem\n\tknative.dev/pkg@v0.0.0-20210331065221-952fdd90dbb0/controller/controller.go:531\nknative.dev/pkg/controller.(*Impl).RunContext.func3\n\tknative.dev/pkg@v0.0.0-20210331065221-952fdd90dbb0/controller/controller.go:468"}
  ```

## Troubleshooting

For basic troubleshooting Service Bindings, please see the troubleshooting guide [here](./docs/troubleshooting.md).

## Samples

Samples are located in the [samples directory](./samples), including:

- [Spring PetClinic with MySQL](./samples/spring-petclinic)
- [Controlled Resource](./samples/controlled-resource)
- [Overridden Type and Provider](./samples/overridden-type-provider)
- [Provisioned Service](./samples/provisioned-service)
- [Multiple Bindings](./samples/multi-binding)

## Resources

### ServiceBinding (servicebinding.io/v1beta1)

The `ServiceBinding` resource shape and behavior is defined by the spec.

```
apiVersion: servicebinding.io/v1beta1
kind: ServiceBinding
metadata:
  name: account-db
spec:
  service:
    apiVersion: bindings.labs.vmware.com/v1alpha1
    kind: ProvisionedService
    name: account-db
  workload:
    apiVersion: apps/v1
    kind: Deployment
    name: account-service
```

### ProvisionedService (bindings.labs.vmware.com/v1alpha1)

The `ProvisionedService` exposes a resource `Secret` by implementing the upstream [Provisioned Service duck type](https://github.com/k8s-service-bindings/spec#provisioned-service), and may be the target of the `.spec.service` reference for a `ServiceBinding`. It is intended for compatibility with existing services that do not directly implement the duck type.

For example to expose a service with an existing `Secret` named `account-db-service`:

```
apiVersion: bindings.labs.vmware.com/v1alpha1
kind: ProvisionedService
metadata:
  name: account-db
spec:
  binding:
    name: account-db-service

---
apiVersion: v1
kind: Secret
metadata:
  name: account-db-service
type: Opaque
stringData:
  type: mysql
  # use appropriate values
  host: localhost
  database: default
  password: ""
  port: "3306"
  username: root
```

The controller writes the resource's status to implement the duck type.

## Contributing

The Service Bindings for Kubernetes project team welcomes contributions from the community. If you wish to contribute code and you have not signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## Acknowledgements

Service Bindings for Kubernetes is an implementation of the [Service Binding Specification for Kubernetes](https://github.com/k8s-service-bindings/spec). Thanks to [Arthur De Magalhaes](https://github.com/arthurdm) and [Ben Hale](https://github.com/nebhale) for leading the spec effort. 

The initial implementation was conceived in [`projectriff/bindings`](https://github.com/projectriff/bindings/) by [Scott Andrews](https://github.com/scothis), [Emily Casey](https://github.com/ekcasey) and the [riff community](https://github.com/orgs/projectriff/people) at large, drawing inspiration from [mattmoor/bindings](https://github.com/mattmoor/bindings) and [Knative](https://knative.dev) duck type reconcilers.

## License

Apache License v2.0: see [LICENSE](./LICENSE) for details.
