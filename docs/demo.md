# `score-cloudrun` Demo!

![](score-cloudrun.png)

Objectives:
- As Developer, I use the Score specification as a contract in order to describe what I need to deploy my workloads.
- As Platform Engineer, I author the Terraform manifests that will be used by `score-cloudrun` to centralize and standardize how the workloads and their infrastructure dependencies will actually be deployed in Cloud Run and associated Google Cloud services.

TOC:
- [As Dev, I define a very basic workload](#as-dev-i-define-a-very-basic-workload)
- [As Dev, I want to know what resource dependencies I can use alongside my workload](#as-dev-i-want-to-know-what-resource-dependencies-i-can-use-alongside-my-workload)
- [As Dev, I now want to use a Redis database](#as-dev-i-now-want-to-use-a-redis-database)
- [As Dev, I now want to use a PostgreSQL database](#as-dev-i-now-want-to-use-a-postgresql-database)
- [As Dev, I now want to expose my workload via a DNS](#as-dev-i-now-want-to-expose-my-workload-via-a-dns)
- [As Dev, I now want to test my workload and dependencies locally](#as-dev-i-now-want-to-test-my-workload-and-dependencies-locally)
- [What's coming next](#whats-coming-next)

```bash
make build
alias score-cloudrun=./score-cloudrun

score-cloudrun init

cp examples/provisioners/* .score-cloudrun
```

## As Dev, I define a very basic workload

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: .
    variables:
      PODINFO_UI_COLOR: "#34577c"
      PODINFO_UI_MESSAGE: ""
```

The associated Cloud Run resources in Terraform can be generated:
```bash
score-cloudrun generate examples/score/score-basic.yaml \
    --location FIXME \
    --project-id FIXME \
    --image ghcr.io/stefanprodan/podinfo:latest \
    -o main.tf
```

And deploy them:
```bash
terraform init

terrraform plan -out out.tfplan

terrraform apply out.tfplan
```

```none
google_cloud_run_v2_service
google_service_account
```

## As Dev, I want to know what resource dependencies I can use alongside my workload

```bash
score-cloudrun provisioners list
```

```none
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
|   TYPE   |  CLASS   |      INPUTS      |                 OUTPUTS                  |                                 DESCRIPTION                                 |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
| dns      | external |                  | host                                     | Makes the Cloud Run Service publicly available                              |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
| dns      | default  |                  | host                                     | Uses the default Cloud Run Service endpoint                                 |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
| postgres | default  |                  | database, host, password, port, username | Generates a Cloud SQL (PostgreSQL) instance and database Terraform resource |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
| redis    | default  |                  | host, password, port, username           | Generates a Memorystore (Redis) Terraform resource                          |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
| route    | default  | host, path, port |                                          | Defines an empty route                                                      |
+----------+----------+------------------+------------------------------------------+-----------------------------------------------------------------------------+
```

## As Dev, I now want to use a Redis database

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: .
    variables:
      PODINFO_UI_COLOR: "#34577c"
      PODINFO_UI_MESSAGE: ""
      REDIS_HOST: ${resources.my-redis.host}
      REDIS_PORT: ${resources.my-redis.port}
      REDIS_USERNAME: ${resources.my-redis.username}
      REDIS_PASSWORD: ${resources.my-redis.password}
resources:
  my-redis:
    type: redis
```

The associated Cloud Run and dependencies resources in Terraform can be generated:
```bash
score-cloudrun generate examples/score/score-redis.yaml \
    --location FIXME \
    --project-id FIXME \
    --image ghcr.io/stefanprodan/podinfo:latest \
    -o main.tf
```

And deploy them:
```bash
terraform init

terrraform plan -out out.tfplan

terrraform apply out.tfplan
```

```none
google_cloud_run_v2_service
google_service_account
google_redis_instance
```

## As Dev, I now want to use a PostgreSQL database

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: .
    variables:
      PODINFO_UI_COLOR: "#34577c"
      PODINFO_UI_MESSAGE: ""
      REDIS_HOST: ${resources.my-redis.host}
      REDIS_PORT: ${resources.my-redis.port}
      REDIS_USERNAME: ${resources.my-redis.username}
      REDIS_PASSWORD: ${resources.my-redis.password}
      POSTGRES_HOST: ${resources.my-postgres.host}
      POSTGRES_DATABASE: ${resources.my-postgres.database}
      POSTGRES_PORT: ${resources.my-postgres.port}
      POSTGRES_USERNAME: ${resources.my-postgres.username}
      POSTGRES_PASSWORD: ${resources.my-postgres.password}
resources:
  my-redis:
    type: redis
  my-postgres:
    type: postgres
```

The associated Cloud Run and dependencies resources in Terraform can be generated:
```bash
score-cloudrun generate examples/score/score-redis-and-postgres.yaml \
    --location FIXME \
    --project-id FIXME \
    --image ghcr.io/stefanprodan/podinfo:latest \
    -o main.tf
```

And deploy them:
```bash
terraform init

terrraform plan -out out.tfplan

terrraform apply out.tfplan
```

```none
google_cloud_run_v2_service
google_service_account
google_sql_database_instance
google_sql_database
google_sql_user
google_redis_instance
```

## As Dev, I now want to expose my workload via a DNS

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: .
    variables:
      PODINFO_UI_COLOR: "#34577c"
      PODINFO_UI_MESSAGE: ""
      REDIS_HOST: ${resources.my-redis.host}
      REDIS_PORT: ${resources.my-redis.port}
      REDIS_USERNAME: ${resources.my-redis.username}
      REDIS_PASSWORD: ${resources.my-redis.password}
      POSTGRES_HOST: ${resources.my-postgres.host}
      POSTGRES_DATABASE: ${resources.my-postgres.database}
      POSTGRES_PORT: ${resources.my-postgres.port}
      POSTGRES_USERNAME: ${resources.my-postgres.username}
      POSTGRES_PASSWORD: ${resources.my-postgres.password}
service:
  ports:
    web:
      port: 8080
      targetPort: 9898
resources:
  my-redis:
    type: redis
  my-postgres:
    type: postgres
  my-dns:
    type: dns
  my-route:
    type: route
    params:
      host: ${resources.my-dns.host}
      path: /
      port: 8080
```

The associated Cloud Run and dependencies resources in Terraform can be generated:
```bash
score-cloudrun generate examples/score/score-redis-and-postgres-and-dns.yaml \
    --location FIXME \
    --project-id FIXME \
    --image ghcr.io/stefanprodan/podinfo:latest \
    -o main.tf
```

And deploy them:
```bash
terraform init

terrraform plan -out out.tfplan

terrraform apply out.tfplan
```

```none
google_cloud_run_v2_service
google_service_account
google_cloud_run_service_iam_policy
google_sql_database_instance
google_sql_database
google_sql_user
google_redis_instance
```

## As Dev, I now want to test my workload and dependencies locally

Same Score file, but now we use `score-compose` to generate the Docker Compose file:
```bash
score-compose init

score-compose generate examples/score/score-redis-and-postgres-and-dns.yaml \
    --image ghcr.io/stefanprodan/podinfo:latest \
    -o compose.yaml
```

And deploy them:
```bash
docker compose up --build -d --remove-orphans
```

```none
CONTAINER ID   IMAGE                                 COMMAND                  CREATED       STATUS                    PORTS                                     NAMES
72fa0a7182ac   mirror.gcr.io/nginx:1-alpine          "/docker-entrypoint.…"   3 hours ago   Up 12 minutes             0.0.0.0:8080->80/tcp, [::]:8080->80/tcp   score-cloudrun-routing-MflvHg-1
83cf4335b8cd   ghcr.io/stefanprodan/podinfo:latest   "./podinfo"              3 hours ago   Up 1 second                                                         score-cloudrun-example-main-1
7e55f78bcf38   mirror.gcr.io/redis:7-alpine          "redis-server /usr/l…"   3 hours ago   Up 12 minutes             6379/tcp                                  score-cloudrun-redis-Li8iSC-1
ae3c395e038b   mirror.gcr.io/postgres:17-alpine      "docker-entrypoint.s…"   3 hours ago   Up 12 minutes (healthy)   5432/tcp                                  score-cloudrun-pg-hNX58v-1
```

## What's coming next

- `score-cloudrun generate --source`
- `score-cloudrun generate --format terraform|config-connector|knative`
- `score-cloudrun init --profiles`
- `score-cloudrun init --provisioners`
- More features like Job, Volumes, Traffic, etc.
- Officially releasing this at KubeCon NA 2025 Atlanta
- We need your feedback!