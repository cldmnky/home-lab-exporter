# syntax=docker/dockerfile:1.4

# ======================
# Build Stage
# ======================
FROM golang:1.24.1 AS builder
RUN arch

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o home-lab-exporter .

# ======================
# Final Image Stage
# ======================
FROM registry.access.redhat.com/ubi10/ubi:latest

LABEL maintainer="magnus@cloudmonkey.org"
LABEL org.opencontainers.image.source="https://github.com/cldmnky/home-lab-exporter"

ENV EXPORTER_LISTEN="0.0.0.0:9191"
ENV UNIFI_URL="https://unifi"
ENV UNIFI_USER="admin"
ENV UNIFI_PASSWORD="password"
ENV REDFISH_TARGET="bmc.mgmt.example.com"
ENV REDFISH_USER="admin"
ENV REDFISH_PASSWORD="password"

# Install Python and pip for redfishtool
RUN dnf install -y python3-pip && \
    pip3 install redfishtool && \
    dnf clean all

# Add app binary
COPY --from=builder /app/home-lab-exporter /usr/local/bin/home-lab-exporter

# Add entrypoint script
COPY <<EOF /usr/local/bin/entrypoint.sh
#!/bin/bash
set -e
echo "Starting Home Lab Exporter..."
echo "Home Lab Exporter needs the following environment variables to be set:"
echo "  - EXPORTER_LISTEN (default: $EXPORTER_LISTEN)"
echo "  - UNIFI_URL (default: $UNIFI_URL)"
echo "  - UNIFI_USER (default: $UNIFI_USER)"
echo "  - UNIFI_PASSWORD (default: $UNIFI_PASSWORD)"
echo "  - REDFISH_TARGET (default: $REDFISH_TARGET)"
echo "  - REDFISH_USER (default: $REDFISH_USER)"
echo "  - REDFISH_PASSWORD (default: $REDFISH_PASSWORD)"
# Load environment variables from the file if it exists
if [ -f /etc/home-lab-exporter/.env ]; then
    echo "Loading environment variables from /etc/home-lab-exporter/.env"
    export $(grep -v '^#' /etc/home-lab-exporter/.env | xargs -d '\n' -r)
else
    echo "No environment file found at /etc/home-lab-exporter.env"
fi


# Only if systemd-creds is available
if [ -d /run/creds ]; then
    export REDFISH_USER="$(< /run/creds/redfish_user 2>/dev/null)"
    export REDFISH_PASSWORD="$(< /run/creds/redfish_password 2>/dev/null)"
    export REDFISH_TARGET="$(< /run/creds/redfish_target 2>/dev/null)"
    export UNIFI_USER="$(< /run/creds/unifi_user 2>/dev/null)"
    export UNIFI_PASSWORD="$(< /run/creds/unifi_password 2>/dev/null)"
    export UNIFI_URL="$(< /run/creds/unifi_url 2>/dev/null)"
else
    echo "No systemd-creds directory found at /run/creds, using environment variables."
fi
if [ ! -z "$EXPORTER_LISTEN" ]; then
    echo "EXPORTER_LISTEN is not set, using default: $EXPORTER_LISTEN"
else
    echo "EXPORTER_LISTEN is set to: $EXPORTER_LISTEN"
fi
if [ -z "$REDFISH_USER" ] || [ -z "$REDFISH_PASSWORD" ] || [ -z "$REDFISH_TARGET" ]; then
    echo "Error: REDFISH_USER, REDFISH_PASSWORD, and REDFISH_TARGET must be set."
    exit 1
fi
if [ -z "$UNIFI_USER" ] || [ -z "$UNIFI_PASSWORD" ] || [ -z "$UNIFI_URL" ]; then
    echo "Error: UNIFI_USER, UNIFI_PASSWORD, and UNIFI_URL must be set."
    exit 1
fi
# Start the Home Lab Exporter
exec /usr/local/bin/home-lab-exporter --listen $EXPORTER_LISTEN
EOF

RUN chmod +x /usr/local/bin/entrypoint.sh
RUN mkdir -p /etc/home-lab-exporter && \
    chown -R 1001:0 /etc/home-lab-exporter && \
    chmod -R g=u /etc/home-lab-exporter

EXPOSE 9191

#USER 1001

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]