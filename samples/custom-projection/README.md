# Custom Projection

[Custom Projection][custom-projection] is an extension to the Service Bindings spec that allows another controller to manage how the binding is applied to the application.
Service binding implementation are not required to support custom projection, so this behavior may not be portable to other implementations.
By default, PodSpecable resources (`Deployment`, `Job`, Knative `Service`, etc) are the only type of resource that the application can reference.

To mark a resource as custom, the `ServiceBinding` resource **MUST** set the `projection.service.binding/type: Custom` annotation.

## Setup

If not already installed, [install the ServiceBinding CRD and controller][install].

## Deploy

Apply the custom application, service and connect them with a ServiceBinding:

```sh
kubectl apply -f ./samples/custom-projection
```

## Understand

It may appear that not much happened.
The normal ServiceBinding reconciler will lookup the service secret, apply mappings to the secret and then project the binding into the application.
With a custom projection, the `ServiceBinding` will not project into the application.
Instead of managing the projection itself, a new resource `ServiceBindingProjection` is created.
It is the responsibility of a third-party reconciler in the cluster to reconcile the `ServiceBindingProjection`, the `ServiceBinding` controller is now hands-off for this binding.

```sh
kubectl describe servicebindingprojections.internal.service.binding custom
```

```
Name:         custom
Namespace:    default
Labels:       service.binding/servicebinding=custom
Annotations:  projection.service.binding/type: Custom
API Version:  internal.service.binding/v1alpha2
Kind:         ServiceBindingProjection
Metadata:
  Creation Timestamp:  2020-07-31T23:14:22Z
  Generation:          1
  Owner References:
    API Version:           service.binding/v1alpha2
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  ServiceBinding
    Name:                  custom
    UID:                   07a4329b-6534-495b-99fa-13c36f8e95b7
  Resource Version:        93219239
  Self Link:               /apis/internal.service.binding/v1alpha2/namespaces/default/servicebindingprojections/custom
  UID:                     80c14f83-969b-47d6-a487-11c62f1ead02
Spec:
  Application:
    API Version:  v1
    Kind:         Secret
    Name:         custom
  Binding:
    Name:  custom-projection
  Name:    custom
Events:    <none>
```

The `ServiceBinding` resource will not become ready until the `ServiceBindingProjection` is successfully reconciled.

## Play

While too advanced for this sample, for those comfortable, try creating a controller to reconcile the `ServiceBindingProjection`.
Make to only reconcile resources with the `projection.service.binding/type: Custom` annotation and an application reference that you know how to process.

If that seems too complex, hopefully the ecosystem will evolve to provide turn-key custom projection reconcilers.
For now, we'll have to wait.

[custom-projection]: https://github.com/k8s-service-bindings/spec/#custom-projection
[install]: ../../README.md#try-it-out
