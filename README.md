# Mock BOP service

This service is used to replace the [BOP](https://github.com/RedHatInsights/backoffice-proxy) service in Ephemeral environments.

It starts up an HTTP server on port `8090`, and forwards requests to different endpoints depending on the URL paths the user requests.

It is meant to be run alongside a [Keycloak](https://www.keycloak.org) service, where it can fetch and print information about users and realms.

## Pre-requisites

While not essential for running this service, you'll need a local Keycloak server with a pre-configured realm named `redhat-external` on it, and some users, with the following attributes:

- `is_active`: type: Boolean
- `is_org_admin`: type: Boolean
- `is_internal`: type: Boolean
- `account_id`: type: String
- `org_id`: type: String
- `entitlements`: type: String
- `account_number`: type: String

## List of supported paths

The current list of supported paths is:

- `/` : Empty endpoint, used as a status check endpoint.
- `/v1/users` : only `POST` requests are allowed. Used to fetch Keycloak users
- `/v1/jwt` : sends a `GET` request against the `KEYCLOAK_SERVER` URL and prints the
              `redhat-external` realm public key
- `/v1/auth` : it expects a basic Authorization Header to be sent with username and
               password, and uses it to request a token from the `redhat-external`
               realm from the `KEYCLOAK_SERVER` URL for that user. Then returns the
               user entity.
- `/v1/accounts` : handles `POST` and `GET` requests for querying users for a specific account
- `/v2/accounts` : expects a GET request with query params defining filters to fetch users on Keycloak
- `/api/entitlements/v1/services` : prints a user's entitlements list based on the
                                    provided Identity header (only Basic Auth is
                                    supported). if the enviroment variable `ALL_PASS`
                                    is found then a fixed JSON object with
                                    entitlements is printed instead.


## How to run

### pre-requisites

You'll need a Keycloak Server running for the Keycloak-related requests to succed,
and pass in the Server URL, Admin username and password using the `KEYCLOAK_SERVER`,
`KEYCLOAK_USERNAME` and `KEYCLOAK_PASSWORD` environment variables respectively.

The current supported version of Keycloak is: `15.0.2` , based on the
[Keycloak version that Clowder uses](https://github.com/RedHatInsights/clowder/blob/983f993067b6ffbed85c7a7a85ee521019f19258/controllers/cloud.redhat.com/providers/web/impl.go#L23)

You will also need a valid Keycloak realm named `redhat-external` as MBOP expects it to be
pre-created. There's a [realm template](./data/redhat-external-realm.json) you can import to
your Keycloak server to help you get started, it defines the redhat-external realm and a test
user.

### How to run Locally

It is recommended if you simply spin a container with a Keycloak server that imports the demo-realm.

You can do it using rootless podman by running:

```shell

podman run -it --name keycloak -p 8080:8080 \
    -e KEYCLOAK_ADMIN_USER=admin \
    -e KEYCLOAK_ADMIN_PASSWORD=change_me \
    -e KEYCLOAK_IMPORT=/opt/keycloak/data/import/redhat-external-realm.json \
    -v ${PWD}/data/redhat-external-realm.json:/opt/keycloak/data/import/redhat-external-realm.json:z \
    quay.io/keycloak/keycloak:15:0.2
```

Then run MBOP, either building and running it locally:

```sh

$ go build ./...

$ KEYCLOAK_SERVER='http://localhost:8080' KEYCLOAK_USERNAME='admin' KEYCLOAK_PASSWORD='change_it' ./mbop
```

Or you can also build the image container and run it locally with podman:

```
podman build -t localhost/mbop:dev .
podman run -it --rm --name mbop -p 8090:8090 -e KEYCLOAK_SERVER='http://localhost:8080' KEYCLOAK_USERNAME='admin' KEYCLOAK_PASSWORD='change_it'  localhost/mbop:dev
```

You can also leverage the provided `podman-compose.yaml` template and run it all together:

**PLEASE NOTE** : if SELINUX is enforced, you have to provide the `:z` label for the bind mount Realm file to import successfully.

```
mv deployments/podman-compose-env deployments/.env

podman-compose -f podman_compose.yaml up -d --build
```

## How to test ?

### Unit tests
Simply run

```
go test ./...
```

### E2E tests

setup a local environment running Keycloak and MBOP:

```
docker-compose -f deployments/compose.yaml up --build -d
```

**NOTE for podman-compose when SELINUX is enforced**: You'll have to set the `SELINUX_LABEL`
environment variable from [deployments/compose.yaml](compose.yaml) to `:z` for the Keycloak
Realm import to succeed. You can use the [deployments/podman-compose-env](podman-compose-env)
file for that purpose.


```
podman-compose -f deployments/compose.yaml up -d
```

you can run Mocha tests with:

```
# npm --prefix test test
```
