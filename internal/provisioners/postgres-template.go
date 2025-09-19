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

const postgresTemplate = `{{ define "postgresTemplate" }}
resource "random_string" "postgres_instance_name" {
  length  = 10
  special = false
  lower   = true
  upper   = false
}

resource "google_sql_database_instance" "instance" {
  name             = "instance-${random_string.memorystore_instance_name.result}"
  region           = "{{ .Location }}"
  database_version = "POSTGRES_14"
  settings {
    tier = "db-g1-small"
  }

  deletion_protection  = true
}

resource "random_string" "postgres_database_name" {
  length  = 10
  special = false
  lower   = true
  upper   = false
}

resource "google_sql_database" "database" {
  name     = "db-${random_string.postgres_database_name.result}"
  instance = google_sql_database_instance.instance.name
  deletion_policy = "ABANDON"
}

resource "random_password" "password" {
  length  = 16
  special = false
}

resource "google_sql_user" "main" {
  instance = google_sql_database_instance.instance.name
  name     = "user-${random_string.postgres_database_name.result}"
  password = random_password.password.result
}{{- end }}
`
