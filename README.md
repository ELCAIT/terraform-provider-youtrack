# youtrack-provider

OpenTofu provider for YouTrack to manage settings, roles, custom fields, and project templates.

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
- SonarQube analysis (when `SONAR_TOKEN` and `SONAR_HOST_URL` are configured)

## Release

GitHub Actions workflow [release.yml](.github/workflows/release.yml) runs on tags matching `v*` and publishes release artifacts through GoReleaser.

Required repository secrets:

- `GPG_PRIVATE_KEY`
- `PASSPHRASE`

Optional for CI SonarQube analysis:

- `SONAR_TOKEN`
- `SONAR_HOST_URL`
