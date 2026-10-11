data "oci_objectstorage_namespace" "ns" {
  compartment_id = var.tenancy_ocid
}

resource "oci_objectstorage_bucket" "tfstate" {
  compartment_id = var.compartment_ocid
  name           = "deskdrop-tfstate"
  namespace      = data.oci_objectstorage_namespace.ns.namespace
}

resource "oci_objectstorage_bucket" "backups" {
  compartment_id = var.compartment_ocid
  name           = "deskdrop-backups"
  namespace      = data.oci_objectstorage_namespace.ns.namespace
}
