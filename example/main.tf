terraform {
  required_providers {
    denobridge = {
      source = "registry.terraform.io/brad-jones/denobridge"
    }
  }
}

provider "denobridge" {}

data "denobridge_datasource" "bjc_mx" {
  path = "${path.module}/providers/datasource_dns_record.ts"
  permissions = {
    all = true
  }

  props = {
    query = "bjc.id.au."
    recordType = "MX"
  }
}

resource "denobridge_resource" "bjc_dnsresults" {
  path = "${path.module}/providers/resource_file.ts"
  permissions = {
    all = true
  }

  props = {
    path = "${path.module}/bjc.mx"
    content = "MX: ${data.denobridge_datasource.bjc_mx.result[0].exchange}"
  }
}

output "mx1" {
  value = data.denobridge_datasource.bjc_mx.result[0].exchange
}
