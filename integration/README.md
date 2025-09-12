# Integration Test Example

This directory contains an example of how to use the SwitchCloud Terraform provider.

## Setup

1. Build the provider:
   ```bash
   cd /workspaces/terraform-provider-switchcloud
   go build -o terraform-provider-switchcloud
   ```

2. Install the provider locally:
   ```bash
   mkdir -p ~/.terraform.d/plugins/local/switchcloud/switchcloud/1.0.0/linux_amd64
   cp terraform-provider-switchcloud ~/.terraform.d/plugins/local/switchcloud/switchcloud/1.0.0/linux_amd64/
   ```

3. Initialize and apply:
   ```bash
   cd integration
   terraform init
   terraform plan
   terraform apply
   ```

Note: This example will fail when actually applied because there's no real SwitchCloud API to connect to. It's intended to demonstrate the configuration syntax and provider structure.
