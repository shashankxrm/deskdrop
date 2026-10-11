# Native OCI Object Storage backend (Terraform >= 1.12). Requires bucket
# deskdrop-tfstate to exist first (chicken-egg). Remaining identity lives in
# backend.hcl (gitignored):
#   terraform init -backend-config=backend.hcl

terraform {
  backend "oci" {
    bucket           = "deskdrop-tfstate"
    key              = "v2/terraform.tfstate"
    auth             = "APIKey"
    private_key_path = "~/.oci/oci_api_key.pem"
  }
}
