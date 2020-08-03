# Environment Variables and Mappings

While life would be easier if provisioned services exposed all of the information necessary to consume the service and if all application understood how to read binding credentials from the volume, there is a lot of existing resources that don't.
The specification defines two concepts to make it easier to support existing services and applications with mappings and environment variables.

Mappings allow for specifying new keys in the projected secret, or composing existing values via a template.
An environment variable maps a specific key in the projected secret into the application's environment.
Together these concepts make it easier to support existing applications.

In this sample, we'll use a [Kubernetes Job][kubernetes-jobs] to dump the environment to the logs and exit.

## Setup

If not already installed, [install the ServiceBinding CRD and controller][install].

## Deploy

Like Pods, Kubernetes Jobs are immutable after they are created.
We need to make sure the `ServiceBinding` is fully configured before the application is created.

Apply the `ProvisionedService` and `ServiceBinding`:

```sh
kubectl apply -f ./samples/environment-variable-mappings/service.yaml -f ./samples/environment-variable-mappings/service-binding.yaml
```

Check on the status of the `ServiceBinding`:

```sh
kubectl get servicebinding mappings -oyaml
```

The `ServiceAvailable` condition should be `True` and the `ProjectionReady` condition `False`.

```
...
  conditions:
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    message: jobs.batch "mappings" not found
    reason: ApplicationMissing
    status: "False"
    type: ProjectionReady
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    message: jobs.batch "mappings" not found
    reason: ApplicationMissing
    status: "False"
    type: Ready
  - lastTransitionTime: "2020-08-03T15:25:45Z"
    status: "True"
    type: ServiceAvailable
```

Create the application `Job`:

```sh
kubectl apply -f ./samples/environment-variable-mappings/application.yaml
```

## Understand

The `ServiceBinding` resources defines both a mapping the combines values from the secret into a new value, and a environment variable that is projected into the application in addition to the binding volume mount.

```yaml
  mappings:
  - name: uri
    value: "https://{{ urlquery .username }}:{{ urlquery .password }}@{{ .host }}"
  env:
  - name: GAME_SERVER
    key: uri
```

Mappings use [Go templates][go-template] to define new keys in the projected secret.
Keys in the secret are exposed to the template as arguments.

The application job dumps the environment to the log and then exits.
We should see our injected environment variable as well as other variable commonly found in Kubernetes containers.

Inspect the logs from the job:

```sh
kubectl logs -l job-name=mappings
```

```
...
GAME_SERVER=https://sfalken:JOSHUA@wopr.norad.mil
...
```

## Play

Try removing and then applying all of the sample resources at once.

```sh
kubectl delete -f ./samples/environment-variable-mappings/
```

```sh
kubectl apply -f ./samples/environment-variable-mappings/
```

The application logs will no longer contain the `GAME_SERVER` variable:

```sh
kubectl logs -l job-name=mappings
```

While technically this is a race condition, it's very unlikely to ever successfully bind to the `Job`, unless the job is created after the `ServiceBinding` is configured.


[install]: ../../README.md#try-it-out
[kubernetes-jobs]: https://kubernetes.io/docs/concepts/workloads/controllers/job/
[go-template]: https://golang.org/pkg/text/template/
