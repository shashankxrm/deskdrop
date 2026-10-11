variable "tenancy_ocid" { type = string }
variable "user_ocid" { type = string }
variable "fingerprint" { type = string }
variable "private_key_path" { type = string }
variable "region" { type = string }
variable "compartment_ocid" { type = string }

variable "availability_domain" { type = string }
variable "instance_ocid" { type = string }
variable "vcn_ocid" { type = string }
variable "subnet_ocid" { type = string }

variable "public_ip" {
  type        = string
  description = "Live public IPv4 (A0: 137.131.14.30). DuckDNS must stay on this."
}

variable "duckdns_domain" {
  type        = string
  description = "Subdomain only, e.g. shashank-dropdesk"
}

variable "duckdns_token" {
  type      = string
  sensitive = true
}

variable "ssh_ingress_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "Tighten to your IP if you want. 80/443 stay 0.0.0.0/0."
}
