# Imported A0 instance. ignore_changes + prevent_destroy so plan cannot
# replace the live Ampere box (created in the console).

resource "oci_core_instance" "deskdrop" {
  availability_domain = var.availability_domain
  compartment_id      = var.compartment_ocid
  display_name        = "deskdrop"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = 1
    memory_in_gbs = 6
  }

  create_vnic_details {
    subnet_id        = var.subnet_ocid
    assign_public_ip = true
  }

  source_details {
    source_type = "image"
    source_id   = "dummy-replaced-on-import"
  }

  lifecycle {
    prevent_destroy = true
    ignore_changes  = all
  }
}
