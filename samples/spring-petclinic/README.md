# Spring PetClinic with MySQL

[Spring PetClinic][petclinic] is a sample [Spring Boot][boot] web application that can be used with MySQL.

## Setup

If not already installed, [install the ServiceBinding CRD and controller][install].

## Deploy

Apply the PetClinic workload, MySQL service and connect them with a ServiceBinding:

```sh
kubectl apply -f ./samples/spring-petclinic
```

Wait for the workload (and database) to start and become healthy:

```sh
kubectl wait deployment spring-petclinic --for condition=available --timeout=2m
```

## Understand

Inspect the PetClinic workload as bound:

```sh
kubectl describe deployment spring-petclinic
```

If the ServiceBinding is working, a new environment variable (SERVICE_BINDING_ROOT), volume and volume mount (binding-49a23274b0590d5057aae1ae621be723716c4dd5) is added to the deployment.
The describe output will contain:

```txt
...
  Containers:
   workload:
    ...
    Environment:
      SPRING_PROFILES_ACTIVE:  mysql
      SERVICE_BINDING_ROOT:    /bindings
    Mounts:
      /bindings/spring-petclinic-db from binding-4b2c350fb984fc36b6cf39515a2efced0fcb5053 (ro)
  Volumes:
   binding-4b2c350fb984fc36b6cf39515a2efced0fcb5053:
    Type:                Projected (a volume that contains injected data from multiple sources)
    SecretName:          spring-petclinic-db
    SecretOptionalName:  <nil>
...
```

The workload uses [Spring Cloud Bindings][scb], which discovers the bound MySQL service by type and reconfigures Spring Boot to consume the service.
Spring Cloud Bindings is automatically added to Spring applications built by Paketo buildpacks.

We can see the effect of Spring Cloud Bindings by view the workload logs:

```sh
kubectl logs -l app=spring-petclinic -c workload --tail 1000
```

The logs should contain:

```txt
...
Spring Cloud Bindings Boot Auto-Configuration Enabled


              |\      _,,,--,,_
             /,`.-'`'   ._  \-;;,_
  _______ __|,4-  ) )_   .;.(__`'-'__     ___ __    _ ___ _______
 |       | '---''(_/._)-'(_\_)   |   |   |   |  |  | |   |       |
 |    _  |    ___|_     _|       |   |   |   |   |_| |   |       | __ _ _
 |   |_| |   |___  |   | |       |   |   |   |       |   |       | \ \ \ \
 |    ___|    ___| |   | |      _|   |___|   |  _    |   |      _|  \ \ \ \
 |   |   |   |___  |   | |     |_|       |   | | |   |   |     |_    ) ) ) )
 |___|   |_______| |___| |_______|_______|___|_|  |__|___|_______|  / / / /
 ==================================================================/_/_/_/

:: Built with Spring Boot :: 2.3.1.RELEASE


2020-07-31 14:48:25.037  INFO 1 --- [           main] o.s.s.petclinic.PetClinicApplication     : Starting PetClinicApplication v2.3.1.BUILD-SNAPSHOT on petclinic-5f5f8ff6db-srn7g with PID 1 (/workspace/BOOT-INF/classes started by cnb in /workspace)
2020-07-31 14:48:25.057  INFO 1 --- [           main] o.s.s.petclinic.PetClinicApplication     : The following profiles are active: mysql
2020-07-31 14:48:25.191  INFO 1 --- [           main] .BindingSpecificEnvironmentPostProcessor : Creating binding-specific PropertySource from Kubernetes Service Bindings
...
```

## Play

To connect to the workload, forward a local port into the cluster:

```sh
kubectl port-forward service/spring-petclinic 8080:80
```

Then open `http://localhost:8080` in a browser.


[petclinic]: https://github.com/spring-projects/spring-petclinic
[boot]: https://spring.io/projects/spring-boot
[paketo]: https://paketo.io
[install]: ../../README.md#try-it-out
[scb]: https://github.com/spring-cloud/spring-cloud-bindings
