terraform {
  required_providers {
    switchcloud = {
      source = "id-unibe-ch/switchcloud"
    }
  }
}

provider "switchcloud" {
  endpoint = "https://cloud.switch.ch"
  api_key  = var.switchcloud_api_key
}

# Create a project
resource "switchcloud_project" "example" {
  name        = "my-project"
  description = "An example project"
}

# Data source to read an existing project
data "switchcloud_project" "existing" {
  name = "existing-project-name"
}

# Variables
variable "switchcloud_api_key" {
  description = "SwitchCloud API Key"
  type        = string
  sensitive   = true
}

# Outputs
output "project_id" {
  value = switchcloud_project.example.id
}

output "existing_project_id" {
  value = data.switchcloud_project.existing.id
}
