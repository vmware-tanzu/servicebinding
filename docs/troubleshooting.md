# Troubleshooting

## Workload is not present or is not in the same namespace

### Symptoms
+ Service bindings logs shows error `Reconcile error`  with message `deployments.apps \"app-name\" not found`
  + `app-name` is the name of the app which the service bindings is looking for

### Cause
+ The service binding is not deployed in the same namespace as the application workload.

### Solution
+ Deploy the workload application in the same namespaces as the `ServiceBinding`.

## Application unable to start due to miss configuration of service namespace

### Symptoms
+ The pod not being able to start due to missing service
+ No errors shown in the Service bindings logs

### Cause
+ The applied binding is empty due to the secret referencing the service is not in the same namespace as the application workload and the `ServiceBinding`.

### Solution
+ Deploy the Binding Secret in the same namespace of the `ServiceBinding`.
