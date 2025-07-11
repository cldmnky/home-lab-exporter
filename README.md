# home-lab-exporter

home-lab-exporter is a Prometheus exporter that collects and exposes metrics from my home lab infrastructure, including:

- Redfish-compatible BMCs (for server hardware monitoring)
- UniFi network controllers (for switch, port, and device metrics)

This exporter enables you to monitor hardware health, network statistics, and other key metrics in your home lab environment using Prometheus and Grafana.

## Environment Variables

The following environment variables must be set for the exporter to function correctly:

- `REDFISH_TARGET` – Redfish BMC address (e.g., `bmc.example.com`)
- `REDFISH_USER` – Redfish username
- `REDFISH_PASSWORD` – Redfish password
- `UNIFI_URL` – UniFi controller URL (e.g., `https://unifi`)
- `UNIFI_USER` – UniFi controller username
- `UNIFI_PASSWORD` – UniFi controller password

You can set these in your environment, a systemd EnvironmentFile, or using systemd-creds for secret management.

Example:
```sh
export REDFISH_TARGET=bmc.example.com
export REDFISH_USER=admin
export REDFISH_PASSWORD=yourpassword
export UNIFI_URL=https://unifi
export UNIFI_USER=youruser
export UNIFI_PASSWORD=yourpassword
```
