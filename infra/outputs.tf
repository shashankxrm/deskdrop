output "public_ip" {
  value = var.public_ip
}

output "instance_id" {
  value = oci_core_instance.deskdrop.id
}

output "nsg_id" {
  value = oci_core_network_security_group.deskdrop.id
}

output "tfstate_bucket" {
  value = oci_objectstorage_bucket.tfstate.name
}

output "backups_bucket" {
  value = oci_objectstorage_bucket.backups.name
}

output "object_storage_namespace" {
  value = data.oci_objectstorage_namespace.ns.namespace
}
