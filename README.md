# terraform-provider-youtrack

OpenTofu provider for YouTrack to manage settings, roles, custom fields, and project templates.

This software is licensed under the Mozilla Public License 2.0 (MPL-2.0).
See the LICENSE file for details.

## Provider Source

Use the provider from the OpenTofu registry as:

```hcl
terraform {
  required_providers {
    youtrack = {
      source = "elcait/youtrack"
    }
  }
}
```

## CI

GitHub Actions workflow [ci.yml](.github/workflows/ci.yml) runs:

- Unit tests with coverage
- golangci-lint
- SonarQube analysis (when `SONAR_TOKEN` is configured)

## Release

GitHub Actions workflow [release.yml](.github/workflows/release.yml) runs on tags matching `v*` and publishes release artifacts through GoReleaser.

Required repository secrets:

- `GPG_PRIVATE_KEY`
- `PASSPHRASE`

Optional for CI SonarQube analysis:

- `SONAR_TOKEN`

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24
- [GolangCI-Lint](https://golangci-lint.run/docs/welcome/install/local/) >= 2.10.1

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Run tests locally

```shell
make test
```

## Generate Documentation

To generate or update documentation, run the following command in the repository directory:

```shell
make generate
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

See documentation for the provider and its resources in the [docs](./docs) directory.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
