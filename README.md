# SwitchCloud Terraform Provider

This Terraform provider enables you to manage SwitchCloud resources using Terraform.

## Features

- **Project Resource**: Create, read, update, and delete SwitchCloud projects
- **Project Data Source**: Read existing SwitchCloud projects

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for development)
- SwitchCloud API access with valid API key

## Installation

### From Source

1. Clone this repository:
   ```bash
   git clone <repository-url>
   cd terraform-provider-switchcloud
   ```

2. Build the provider:
   ```bash
   go build
   ```

3. Install locally for development:
   ```bash
   mkdir -p ~/.terraform.d/plugins/local/switchcloud/switchcloud/1.0.0/linux_amd64
   cp terraform-provider-switchcloud ~/.terraform.d/plugins/local/switchcloud/switchcloud/1.0.0/linux_amd64/
   ```

## Provider Configuration

```hcl
terraform {
  required_providers {
    switchcloud = {
      source = "local/switchcloud/switchcloud"
    }
  }
}

provider "switchcloud" {
  endpoint = "https://api.switchcloud.com"  # Optional, defaults to this
  api_key  = var.switchcloud_api_key        # Required for authentication
}
```

### Configuration Arguments

- `endpoint` (Optional) - The SwitchCloud API endpoint. Defaults to `https://api.switchcloud.com`
- `api_key` (Optional) - SwitchCloud API key for authentication. Can also be set via environment variable `SWITCHCLOUD_API_KEY`

## Resources

### `switchcloud_project`

Manages a SwitchCloud project.

#### Example Usage

```hcl
resource "switchcloud_project" "example" {
  name            = "my-project"
  description     = "An example project"
  organisation_id = "org-123456"
}
```

#### Argument Reference

- `name` (Required) - The name of the project
- `description` (Optional) - A description of the project
- `organisation_id` (Required) - The ID of the organisation that owns this project

#### Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The unique identifier of the project
- `archived` - Whether the project is archived
- `archived_at` - When the project was archived (if applicable)
- `created_at` - When the project was created
- `updated_at` - When the project was last updated

## Data Sources

### `switchcloud_project`

Reads information about an existing SwitchCloud project.

#### Example Usage

```hcl
data "switchcloud_project" "example" {
  id = "project-123456"
}
```

#### Argument Reference

- `id` (Required) - The unique identifier of the project

#### Attribute Reference

- `name` - The name of the project
- `description` - The description of the project
- `organisation_id` - The ID of the organisation that owns this project
- `archived` - Whether the project is archived
- `archived_at` - When the project was archived (if applicable)
- `created_at` - When the project was created
- `updated_at` - When the project was last updated

## API Endpoints

The provider makes the following API calls:

- `POST /api/v1/projects` - Create a new project
- `GET /api/v1/projects/{id}` - Read a project
- `PUT /api/v1/projects/{id}` - Update a project
- `DELETE /api/v1/projects/{id}` - Delete a project

## Authentication

The provider supports authentication via API key passed in the `Authorization: Bearer <api_key>` header.

## Development

### Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

### Running Tests

```bash
go test ./...
```

### Running Acceptance Tests

```bash
TF_ACC=1 go test ./internal/provider -v
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes
5. Run tests and ensure they pass
6. Submit a pull request

## License

This project is licensed under the MPL-2.0 License - see the LICENSE file for details.

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
