// Copyright 2025 The Score Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioners

const redisTemplate = `{{ define "redisTemplate" }}
variable "network" {
  type    = string
  default = "default"
}

resource "random_string" "memorystore_instance_name" {
  length  = 10
  special = false
  lower   = true
  upper   = false
}

resource "google_redis_instance" "memorystore" {
  name               = "redis-${random_string.memorystore_instance_name.result}"
  memory_size_gb     = 1
  redis_version      = "REDIS_7_2"
  region             = "{{ .Location }}"
  auth_enabled       = true
  authorized_network = data.google_compute_network.vpc.id
}

data "google_compute_network" "vpc" {
  name = var.network
}{{- end }}
`
