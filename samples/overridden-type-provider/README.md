# Overridden Type and Provider

When projected into the application workload, the binding must contain a `type` entry and should contain a `provider` entry.
If the Secret doesn't contain a type or provider, or contains the wrong values, they can be overridden for the binding.

In this sample, we'll use a [Kubernetes Job][kubernetes-jobs] to dump the environment to the logs and exit.

## Setup

If not already installed, [install the ServiceBinding CRD and controller][install].

## Deploy

Like Pods, Kubernetes Jobs are immutable after they are created.
We need to make sure the `ServiceBinding`s are fully configured before the application is created.

Apply the `ProvisionedService` and `ServiceBinding`:

```sh
kubectl apply -f ./samples/overridden-type-provider/service.yaml -f ./samples/overridden-type-provider/service-binding.yaml
```

Check on the status of the `ServiceBinding`:

```sh
kubectl get servicebinding -l sample=overridden-type-provider -oyaml
```

For each service binding, the `ServiceAvailable` condition should be `True` and the `ProjectionReady` condition `False`.

```
...
  conditions:
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    message: jobs.batch "overridden-type-provider" not found
    reason: ApplicationMissing
    status: "False"
    type: ProjectionReady
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    message: jobs.batch "overridden-type-provider" not found
    reason: ApplicationMissing
    status: "False"
    type: Ready
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    status: "True"
    type: ServiceAvailable
```

Create the application `Job`:

```sh
kubectl apply -f ./samples/overridden-type-provider/application.yaml
```

## Understand

Each `ServiceBinding` resource defines an environment variable that is projected into the application in addition to the binding volume mount.

```sh
kubectl describe job overridden-type-provider
```

```
...
Environment:
  SERVICE_BINDING_ROOT:  /bindings
  BOUND_PROVIDER:         (v1:metadata.annotations['internal.bindings.labs.vmware.com/projection-717760b0e7853de4a23bffadf9d02d6109c6ad2e-provider'])
  BOUND_TYPE:             (v1:metadata.annotations['internal.bindings.labs.vmware.com/projection-717760b0e7853de4a23bffadf9d02d6109c6ad2e-type'])
...
```

The application job dumps the environment to the log and then exits.
We should see our injected environment variable as well as other variable commonly found in Kubernetes containers.

Inspect the logs from the job:

```sh
kubectl logs -l job-name=overridden-type-provider
```

```
...
SERVICE_BINDING_ROOT=/bindings
BOUND_PROVIDER=overridden-provider
BOUND_TYPE=overridden-type
...
```

## Play

Try changing the `.spec.type` or `.spec.provider` fields on the ServiceBinding, or return them to the original values (empty string).
Remember that Jobs are immutable after they are created, so you'll need to delete and recreate the Job to see the additional binding.

Alternatively, define a `Deployment` and update each binding to target the new Deployment.
Since Deployments are mutable, each service binding that is added or removed will be reflected on the Deployment and trigger the rollout of a new `ReplicaSet`.

[install]: ../../README.md#try-it-out
[kubernetes-jobs]: https://kubernetes.io/docs/concepts/workloads/controllers/job/
