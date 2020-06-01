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
    network = var.network_name
    access_config {
      // Ephemeral IP
    }
  }

  metadata = {
    foo = "bar"
  }
}