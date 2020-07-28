
# Service Bindings for Kubernetes

Service Bindings for Kubernetes implements the [Service Binding Specification for Kubernetes](https://github.com/k8s-service-bindings/spec). We are tracking changes to the spec as it approaches a stable release (currently targeting [RC2](https://github.com/k8s-service-bindings/spec/tree/1.0.0-rc2)). Backwards and forwards compatibility should not be expected for alpha versioned resources.

## Try it out

Using the [latest release](https://github.com/vmware-labs/service-bindings/releases/latest) is recommended.

### Build from source

We use [`ko`](https://github.com/google/ko) to build and deploy the CRD and reconciler.

From within the cloned directory for this project, run:

```
ko apply --strict -f config
```

## Resources

### ServiceBinding (service.binding/v1alpha2)

The `ServiceBinding` resource shape and behavior is defined upstream. In addition to the [core specification](https://github.com/k8s-service-bindings/spec#service-binding), the [Role-Based Access Control (RBAC) extension](https://github.com/k8s-service-bindings/spec#role-based-access-control-rbac) is also supported by this implementation.

```
apiVersion: service.binding/v1alpha2
kind: ServiceBinding
metadata:
  name: account-db
spec:
  application:
    apiVersion: apps/v1
    kind: Deployment
    name: account-service
  service:
    apiVersion: bindings.labs.vmware.com/v1alpha1
    kind: ProvisionedService
    name: account-db
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
