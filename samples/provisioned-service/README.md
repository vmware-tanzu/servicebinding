# Provisioned Service

In managed environments, developers typically will not interact directly with a `Secret` they created to connect to a service.
Instead, the `Secret` is provided to them.
The Service Binding Specification defines a [Provisioned Service][provisioned-service] duck-type for any resource to be consumed by a binding.
The target service needs to define `.status.binding.name` referencing the `Secret` containing the binding.

This implementation provides a `ProvisionedService` resource that conforms to the duck-type.
Real services should implement the duck-type directly instead of using this custom resource.

This sample will not be portable across other service binding implementations.

## Setup

If not already installed, [install the ServiceBinding CRD and controller][install].

## Deploy

Apply the custom application, service and connect them with a `ServiceBinding`:

```sh
kubectl apply -f ./samples/provisioned-service
```

## Understand

The application.yaml defines a Kubernetes `Service` and `Deployment` representing our workload.
The service.yaml defined a `ProvisionedService` exposing it's `.status.binding.name` referencing a `Secret`.
The `ServiceBinding`'s service reference target the `ProvisionedService`.

```sh
kubectl describe provisionedservices.bindings.labs.vmware.com provisioned-service
```

Will expose the "provisioned-service" `Secret` on its status.

```
...
Status:
  Binding:
    Name:  provisioned-service
...
```

## Play

Try invoking the Deployment to view the message injected by the service binding.

First, forward a local port into the cluster:

```sh
kubectl port-forward svc/provisioned-service 8080
```

Then, make a request to the service.

```sh
curl http://localhost:8080
Hello service binding!
```

Going further, try updating the `ProvisionedService` to point at a different secret. The service binding is now decoupled from knowledge of a specific `Secret`. Be aware that while the value in the projected secret will be kept up to date, the application uses environment variables that will require the pod be restarted to detect a new value.


[provisioned-service]: https://github.com/k8s-service-bindings/spec/#provisioned-service
[install]: ../../README.md#try-it-out
