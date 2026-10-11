resource "oci_core_vcn" "deskdrop" {
  compartment_id = var.compartment_ocid
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "deskdrop"
  dns_label      = "deskdrop"

  lifecycle {
    prevent_destroy = true
    ignore_changes  = all
  }
}

resource "oci_core_subnet" "public" {
  cidr_block     = "10.0.0.0/24"
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.deskdrop.id
  display_name   = "public"
  dns_label      = "public"

  lifecycle {
    prevent_destroy = true
    ignore_changes  = all
  }
}

resource "oci_core_network_security_group" "deskdrop" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.deskdrop.id
  display_name   = "deskdrop"
}

resource "oci_core_network_security_group_security_rule" "ssh" {
  network_security_group_id = oci_core_network_security_group.deskdrop.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = var.ssh_ingress_cidr
  source_type               = "CIDR_BLOCK"
  description               = "ssh"

  tcp_options {
    destination_port_range {
      min = 22
      max = 22
    }
  }
}

resource "oci_core_network_security_group_security_rule" "http" {
  network_security_group_id = oci_core_network_security_group.deskdrop.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"
  description               = "http"

  tcp_options {
    destination_port_range {
      min = 80
      max = 80
    }
  }
}

resource "oci_core_network_security_group_security_rule" "https" {
  network_security_group_id = oci_core_network_security_group.deskdrop.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"
  description               = "https"

  tcp_options {
    destination_port_range {
      min = 443
      max = 443
    }
  }
}
