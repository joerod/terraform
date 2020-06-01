resource "google_compute_firewall" "default" {
  name    = "default-allow-winrm"
  network = google_compute_network.default.name

  allow {
    protocol = "tcp"
    ports    = ["5985"]
  }
}

resource "google_compute_network" "default" {
  name = var.network_name
}