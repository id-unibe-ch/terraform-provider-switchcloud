terraform {
  required_providers {
    switchcloud = {
      source = "local/switchcloud/switchcloud"
    }
  }
}

provider "switchcloud" {
  endpoint = "https://api.switchcloud.com"
  api_key  = var.switchcloud_api_key
}

variable "switchcloud_api_key" {
  description = "SwitchCloud API Key"
  type        = string
  sensitive   = true
  default     = "test-api-key"
}

# Create a new project
resource "switchcloud_project" "main" {
  name        = "terraform-managed-project"
  description = "A project managed by Terraform"
}

# Create another project with minimal configuration
resource "switchcloud_project" "minimal" {
  name = "minimal-project"
}

# Data source to read an existing project
data "switchcloud_project" "existing" {
  id = "existing-project-id"
}

# Outputs
output "main_project_id" {
  description = "ID of the main project"
  value       = switchcloud_project.main.id
}

output "main_project_created_at" {
  description = "Creation timestamp of the main project"
  value       = switchcloud_project.main.created_at
}

output "minimal_project_id" {
  description = "ID of the minimal project"
  value       = switchcloud_project.minimal.id
}

output "existing_project_info" {
  description = "Information about the existing project"
  value = {
    name            = data.switchcloud_project.existing.name
    description     = data.switchcloud_project.existing.description
    organisation_id = data.switchcloud_project.existing.organisation_id
    archived        = data.switchcloud_project.existing.archived
    created_at      = data.switchcloud_project.existing.created_at
    updated_at      = data.switchcloud_project.existing.updated_at
  }
}
