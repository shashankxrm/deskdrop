resource "terraform_data" "duckdns" {
  triggers_replace = [
    var.public_ip,
    var.duckdns_domain,
  ]

  provisioner "local-exec" {
    command = "curl -fsS \"https://www.duckdns.org/update?domains=${var.duckdns_domain}&token=${var.duckdns_token}&ip=${var.public_ip}&verbose=true\""
  }
}
