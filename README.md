# Aruba Instant On Exporter

English | [日本語](README_ja.md)

A Prometheus exporter for Aruba Instant On network infrastructure, providing comprehensive monitoring metrics for sites, devices, and connected clients.

## Features

- **Site Monitoring**: Track site health, status, and configuration
- **Device Metrics**: Monitor access points, switches, and other network devices
- **Client Analytics**: Track wireless and wired client connections
- **Network Performance**: Monitor signal quality, uptime, and connection statistics
- **Prometheus Integration**: Native Prometheus metrics format for easy integration

## Metrics Exported

### Site Metrics

#### `aruba_instant_on_sites_total`
- **Type**: Gauge
- **Description**: Total number of sites under management
- **Labels**: None

#### `aruba_instant_on_site_info`
- **Type**: Gauge
- **Description**: Site detailed information (value is always 1)
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name
  - `health`: Site health status (Good/Fair/Poor)
  - `status`: Site status (Up/Down)
  - `timezone`: Site timezone

### Device Metrics

#### `aruba_instant_on_devices_total`
- **Type**: Gauge
- **Description**: Total number of devices per site
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name

#### `aruba_instant_on_device_info`
- **Type**: Gauge
- **Description**: Device detailed information (value is always 1)
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name
  - `device_id`: Unique device identifier
  - `device_name`: Device name
  - `device_type`: Device type (accessPoint/switch etc.)
  - `model`: Device model
  - `serial_number`: Serial number
  - `mac_address`: MAC address
  - `ip_address`: IP address
  - `status`: Device status
  - `operational_state`: Operational state

#### `aruba_instant_on_device_uptime_seconds`
- **Type**: Gauge
- **Description**: Device uptime in seconds
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name
  - `device_id`: Unique device identifier
  - `device_name`: Device name

### Client Metrics

#### `aruba_instant_on_wireless_clients_total`
- **Type**: Gauge
- **Description**: Total number of wireless clients per site
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name

#### `aruba_instant_on_wired_clients_total`
- **Type**: Gauge
- **Description**: Total number of wired clients per site (currently not implemented)
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name

#### `aruba_instant_on_clients_by_network`
- **Type**: Gauge
- **Description**: Number of clients by SSID/network
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name
  - `network_ssid`: Network SSID name

#### `aruba_instant_on_clients_by_ap`
- **Type**: Gauge
- **Description**: Number of clients per access point
- **Labels**:
  - `site_id`: Unique site identifier
  - `site_name`: Site name
  - `device_id`: Access point device ID
  - `device_name`: Access point name

## Installation

### Prerequisites

- Go 1.21 or later
- Valid Aruba Instant On account credentials
- Access to Aruba Instant On cloud portal

### From Source

```bash
git clone https://github.com/csenet/instanton-exporter.git
cd instanton-exporter
go build -o instanton-exporter .
```

### Docker

Pull from GitHub Container Registry:

```bash
docker pull ghcr.io/csenet/instanton-exporter:latest
```

Run with environment variables:

```bash
docker run -d \
  --name instanton-exporter \
  -p 9100:9100 \
  -e ARUBA_USERNAME=your-email@example.com \
  -e ARUBA_PASSWORD=your-password \
  ghcr.io/csenet/instanton-exporter:latest
```

### Docker Compose with Full Monitoring Stack

For a complete monitoring setup with Prometheus and Grafana:

1. Copy the environment file:
```bash
cp .env.example .env
```

2. Edit `.env` with your credentials:
```bash
ARUBA_USERNAME=your-email@example.com
ARUBA_PASSWORD=your-password
GRAFANA_PASSWORD=admin
```

3. Start the stack:
```bash
docker-compose up -d
```

This provides:
- **Instanton Exporter** on port 9100 (metrics)
- **Prometheus** on port 9090 (collection)
- **Grafana** on port 3000 (visualization)

### Simple Docker Compose

Or use a minimal docker-compose:

```yaml
version: '3.8'
services:
  instanton-exporter:
    image: ghcr.io/csenet/instanton-exporter:latest
    ports:
      - "9100:9100"
    environment:
      - ARUBA_USERNAME=your-email@example.com
      - ARUBA_PASSWORD=your-password
    restart: unless-stopped
```

## Configuration

### Environment Variables

The exporter requires the following environment variables:

- `ARUBA_USERNAME` - Your Aruba Instant On account email
- `ARUBA_PASSWORD` - Your Aruba Instant On account password

### Using .env File

Create a `.env` file in the project directory:

```env
ARUBA_USERNAME=user@example.com
ARUBA_PASSWORD=your-secure-password
```

### Command Line

```bash
export ARUBA_USERNAME="user@example.com"
export ARUBA_PASSWORD="your-secure-password"
./instanton-exporter
```

## Usage

1. Set up your credentials (see Configuration section)
2. Run the exporter:
   ```bash
   ./instanton-exporter
   ```
3. The exporter will start on port `9100` by default
4. Metrics are available at `http://localhost:9100/metrics`

### Sample Output

```
# HELP aruba_instant_on_sites_total Total number of sites
# TYPE aruba_instant_on_sites_total gauge
aruba_instant_on_sites_total 2

# HELP aruba_instant_on_device_uptime_seconds Device uptime in seconds
# TYPE aruba_instant_on_device_uptime_seconds gauge
aruba_instant_on_device_uptime_seconds{device_id="...",device_name="Office-AP",site_id="...",site_name="Main Office"} 86400

# HELP aruba_instant_on_wireless_clients_total Total number of wireless clients
# TYPE aruba_instant_on_wireless_clients_total gauge
aruba_instant_on_wireless_clients_total{site_id="...",site_name="Main Office"} 15
```

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'aruba-instant-on'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 30s
    scrape_timeout: 10s
```

## Grafana Dashboard

The metrics can be visualized using Grafana. Key dashboard panels might include:

- Site health overview
- Device status and uptime
- Client connection trends
- Network utilization by SSID
- Access point performance

## Authentication

The exporter uses Aruba's OAuth 2.0 with PKCE (Proof Key for Code Exchange) for secure authentication:

1. Fetches dynamic settings from Aruba portal
2. Performs MFA validation to get session token
3. Exchanges session token for authorization code using PKCE
4. Obtains access token for API calls

## API Rate Limiting

The exporter collects metrics every 30 seconds by default. Aruba Instant On APIs have rate limits, so avoid setting collection intervals too aggressively.

## Development

### Project Structure

```
├── main.go              # Main application and Prometheus metrics
├── auth/                # Authentication handling
│   ├── client.go        # OAuth2/PKCE authentication
│   └── pkce.go          # PKCE implementation
├── models/              # Data structures
│   └── settings.go      # Configuration models
├── go.mod               # Go module definition
└── .env.example         # Environment configuration template
```

### Building

```bash
go mod tidy
go build -o instanton-exporter .
```

### Testing

```bash
go test ./...
```

## Security Considerations

- Store credentials securely (environment variables, secrets management)
- The exporter uses HTTPS for all API communications
- OAuth2 tokens are automatically refreshed as needed
- No credentials are logged or exposed in metrics

## Troubleshooting

### Authentication Issues

- Verify your Aruba Instant On credentials
- Ensure your account has access to the sites you want to monitor
- Check for any MFA requirements on your account

### Network Issues

- Ensure outbound HTTPS access to `portal.instant-on.hpe.com`
- Check firewall rules for the exporter's listening port (9100)

### Debugging

Enable debug logging by uncommenting debug statements in the source code and rebuilding.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This is an unofficial exporter and is not affiliated with or supported by HPE/Aruba. Use at your own risk.

## Support

- Create an issue for bug reports or feature requests
- Check existing issues before creating new ones
- Include logs and configuration details when reporting issues

---

Built with ❤️ for the network monitoring community
