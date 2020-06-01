provider "google" {
  credentials = file("C:/Users/joero/Downloads/infra-278803-da1e6f8f7370.json")
  project     = "infra-278803"
  region      = "us-east4"
}

resource "google_compute_instance" "default" {
  name         = "test"
  machine_type = "e2-standard-4"
  zone         = "us-east4-c"
  tags = ["test", "joerod"]

  boot_disk {
    initialize_params {
      image = "windows-sql-cloud/sql-std-2019-win-2019"
    }
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral IP
    }
  }

  metadata = {
    foo = "bar"
  }
}