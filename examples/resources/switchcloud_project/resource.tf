resource "switchcloud_project" "example" {
  name              = "my-project"
  description       = "An example project created via Terraform"
  billing_reference = "REF-1234"
}
