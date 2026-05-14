package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/csenet/instanton-exporter/auth"
)

type ArubaClient struct {
	authClient *auth.Client
	httpClient *http.Client
	baseURL    string
	apiVersion string
}

func NewArubaClient(username, password string) *ArubaClient {
	return &ArubaClient{
		authClient: auth.NewClient(username, password),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    "https://portal.instant-on.hpe.com/api",
		apiVersion: "24",
	}
}

func (c *ArubaClient) Request(method, endpoint string, body io.Reader) (*http.Response, error) {
	token, err := c.authClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	fullURL := c.baseURL + endpoint
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ion-api-version", c.apiVersion)
	req.Header.Set("x-ion-client-platform", "web")
	req.Header.Set("x-ion-client-type", "InstantOn")

	resp, err := c.httpClient.Do(req)

	// If we get a 401, the token might have been invalidated - try once more with a fresh token
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		log.Printf("[WARN] Received 401 Unauthorized, forcing token refresh...")

		// Force token refresh by calling GetAccessToken directly
		if err := c.authClient.GetAccessToken(); err != nil {
			return nil, fmt.Errorf("failed to refresh access token after 401: %w", err)
		}

		// Retry the request with the new token
		token, err = c.authClient.GetToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get refreshed access token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.httpClient.Do(req)
	}

	return resp, err
}

type Site struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Health   string `json:"health"`
	Status   string `json:"status"`
	TimeZone string `json:"timezoneIana"`
}

type Device struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DeviceType       string `json:"deviceType"`
	Model            string `json:"model"`
	SerialNumber     string `json:"serialNumber"`
	MacAddress       string `json:"macAddress"`
	IPAddress        string `json:"ipAddress"`
	Status           string `json:"status"`
	OperationalState string `json:"operationalState"`
	UptimeInSeconds  int    `json:"uptimeInSeconds"`
}

type InventoryResponse struct {
	TotalCount int      `json:"totalCount"`
	Elements   []Device `json:"elements"`
}

type WirelessClient struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	HostName                    string `json:"hostName"`
	ClientType                  string `json:"clientType"`
	WirelessNetworkName         string `json:"wirelessNetworkName"`
	WirelessNetworkId           string `json:"wirelessNetworkId"`
	IPAddress                   string `json:"ipAddress"`
	MacAddress                  string `json:"macAddress"`
	DeviceName                  string `json:"deviceName"`
	DeviceId                    string `json:"deviceId"`
	ConnectionDurationInSeconds int    `json:"connectionDurationInSeconds"`
	Health                      string `json:"health"`
	Status                      string `json:"status"`
	WirelessBand                string `json:"wirelessBand"`
	SignalQuality               string `json:"signalQuality"`
	SignalInDbm                 int    `json:"signalInDbm"`
	SnrInDb                     int    `json:"snrInDb"`
}

type ClientSummaryResponse struct {
	TotalCount int              `json:"totalCount"`
	Elements   []WirelessClient `json:"elements"`
}

type SitesResponse struct {
	TotalCount int    `json:"totalCount"`
	Elements   []Site `json:"elements"`
}

type LandingPage struct {
	Kind                                         string `json:"kind"`
	WirelessClientsCount                         int    `json:"wirelessClientsCount"`
	WiredClientsCount                            int    `json:"wiredClientsCount"`
	ConfiguredNetworksCount                      int    `json:"configuredNetworksCount"`
	CurrentlyActiveNetworksCount                 int    `json:"currentlyActiveNetworksCount"`
	ConfiguredWiredNetworksCount                 int    `json:"configuredWiredNetworksCount"`
	CurrentlyActiveWiredNetworksCount            int    `json:"currentlyActiveWiredNetworksCount"`
	ConfiguredWirelessNetworksCount              int    `json:"configuredWirelessNetworksCount"`
	CurrentlyActiveWirelessNetworksCount         int    `json:"currentlyActiveWirelessNetworksCount"`
	CurrentNetworkThroughputInBitsPerSecond      int64  `json:"currentNetworkThroughputInBitsPerSecond"`
	TotalDataTransferredDuringLast24HoursInBytes int64  `json:"totalDataTransferredDuringLast24HoursInBytes"`
	Health                                       string `json:"health"`
	HealthReason                                 string `json:"healthReason"`
	ActiveAlertsCount                            int    `json:"activeAlertsCount"`
	DeviceCount                                  int    `json:"deviceCount"`
	DeviceUpCount                                int    `json:"deviceUpCount"`
}

type AlertsSummary struct {
	Kind                   string `json:"kind"`
	ActiveInfoAlertsCount  int    `json:"activeInfoAlertsCount"`
	ActiveMinorAlertsCount int    `json:"activeMinorAlertsCount"`
	ActiveMajorAlertsCount int    `json:"activeMajorAlertsCount"`
}

// ProbeEndpoint calls an arbitrary endpoint, pretty-prints the JSON response,
// and returns the status code plus body. Used by PROBE=1 mode to discover the
// shape of undocumented endpoints before writing typed structs against them.
func (c *ArubaClient) ProbeEndpoint(endpoint string) (int, string, error) {
	resp, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		return resp.StatusCode, pretty.String(), nil
	}
	return resp.StatusCode, string(body), nil
}

func (c *ArubaClient) GetSites() (*SitesResponse, error) {
	resp, err := c.Request("GET", "/sites/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var sitesResp SitesResponse
	if err := json.Unmarshal(body, &sitesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sitesResp, nil
}

func (c *ArubaClient) GetInventory(siteID string) (*InventoryResponse, error) {
	resp, err := c.Request("GET", "/sites/"+siteID+"/inventory", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var inventoryResp InventoryResponse
	if err := json.Unmarshal(body, &inventoryResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &inventoryResp, nil
}

func (c *ArubaClient) GetClientSummary(siteID string) (*ClientSummaryResponse, error) {
	resp, err := c.Request("GET", "/sites/"+siteID+"/clientSummary", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get client summary: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var clientResp ClientSummaryResponse
	if err := json.Unmarshal(body, &clientResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &clientResp, nil
}

func (c *ArubaClient) GetLandingPage(siteID string) (*LandingPage, error) {
	resp, err := c.Request("GET", "/sites/"+siteID+"/landingPage", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get landing page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var lp LandingPage
	if err := json.Unmarshal(body, &lp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &lp, nil
}

func (c *ArubaClient) GetAlertsSummary(siteID string) (*AlertsSummary, error) {
	resp, err := c.Request("GET", "/sites/"+siteID+"/alertsSummary", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts summary: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var as AlertsSummary
	if err := json.Unmarshal(body, &as); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &as, nil
}

var (
	sitesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_sites_total",
			Help: "Total number of sites",
		},
		[]string{},
	)

	siteInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_info",
			Help: "Site information",
		},
		[]string{"site_id", "site_name", "health", "status", "timezone"},
	)

	devicesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_devices_total",
			Help: "Total number of devices",
		},
		[]string{"site_id", "site_name"},
	)

	deviceInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_device_info",
			Help: "Device information",
		},
		[]string{"site_id", "site_name", "device_id", "device_name", "device_type", "model", "serial_number", "mac_address", "ip_address", "status", "operational_state"},
	)

	deviceUptime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_device_uptime_seconds",
			Help: "Device uptime in seconds",
		},
		[]string{"site_id", "site_name", "device_id", "device_name"},
	)

	wirelessClientsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_wireless_clients_total",
			Help: "Total number of wireless clients",
		},
		[]string{"site_id", "site_name"},
	)

	wiredClientsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_wired_clients_total",
			Help: "Total number of wired clients",
		},
		[]string{"site_id", "site_name"},
	)

	clientsByNetwork = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_clients_by_network",
			Help: "Number of clients by network SSID",
		},
		[]string{"site_id", "site_name", "network_ssid"},
	)

	clientsByAP = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_clients_by_ap",
			Help: "Number of clients by access point",
		},
		[]string{"site_id", "site_name", "device_id", "device_name"},
	)

	siteThroughputBps = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_throughput_bits_per_second",
			Help: "Current aggregate network throughput for the site in bits per second",
		},
		[]string{"site_id", "site_name"},
	)

	siteData24hBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_data_transferred_24h_bytes",
			Help: "Total bytes transferred at the site during the last 24 hours",
		},
		[]string{"site_id", "site_name"},
	)

	siteDevicesUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_devices_up",
			Help: "Number of devices reporting up at the site",
		},
		[]string{"site_id", "site_name"},
	)

	siteActiveAlerts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_active_alerts",
			Help: "Number of active (uncleared) alerts at the site, broken down by severity",
		},
		[]string{"site_id", "site_name", "severity"},
	)

	siteNetworksConfigured = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_networks_configured",
			Help: "Number of networks configured at the site (by kind: wired/wireless/total)",
		},
		[]string{"site_id", "site_name", "kind"},
	)

	siteNetworksActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aruba_instant_on_site_networks_active",
			Help: "Number of networks currently active at the site (by kind: wired/wireless/total)",
		},
		[]string{"site_id", "site_name", "kind"},
	)
)

type Collector struct {
	client *ArubaClient
}

func NewCollector(client *ArubaClient) *Collector {
	return &Collector{
		client: client,
	}
}

func (c *Collector) Collect() {
	sites, err := c.client.GetSites()
	if err != nil {
		log.Printf("Failed to get sites: %v", err)
		return
	}

	sitesTotal.WithLabelValues().Set(float64(sites.TotalCount))

	for _, site := range sites.Elements {
		siteInfo.WithLabelValues(
			site.ID,
			site.Name,
			site.Health,
			site.Status,
			site.TimeZone,
		).Set(1)

		// Get devices for this site
		inventory, err := c.client.GetInventory(site.ID)
		if err != nil {
			log.Printf("Failed to get inventory for site %s: %v", site.Name, err)
			continue
		}

		devicesTotal.WithLabelValues(site.ID, site.Name).Set(float64(inventory.TotalCount))

		for _, device := range inventory.Elements {
			deviceInfo.WithLabelValues(
				site.ID,
				site.Name,
				device.ID,
				device.Name,
				device.DeviceType,
				device.Model,
				device.SerialNumber,
				device.MacAddress,
				device.IPAddress,
				device.Status,
				device.OperationalState,
			).Set(1)

			deviceUptime.WithLabelValues(
				site.ID,
				site.Name,
				device.ID,
				device.Name,
			).Set(float64(device.UptimeInSeconds))
		}

		// Get wireless clients for this site. The TotalCount field of this
		// endpoint is broken under x-ion-api-version: 24 (always returns 0),
		// so wirelessClientsTotal is populated from landingPage above instead.
		// We still call this to count clients by network SSID and by AP.
		wirelessClients, err := c.client.GetClientSummary(site.ID)
		if err != nil {
			log.Printf("Failed to get wireless clients for site %s: %v", site.Name, err)
		} else {
			// Count clients by network SSID
			networkCounts := make(map[string]int)
			for _, client := range wirelessClients.Elements {
				networkCounts[client.WirelessNetworkName]++
			}
			for ssid, count := range networkCounts {
				clientsByNetwork.WithLabelValues(site.ID, site.Name, ssid).Set(float64(count))
			}

			// Count clients by access point
			apCounts := make(map[string]struct {
				DeviceId   string
				DeviceName string
				Count      int
			})
			for _, client := range wirelessClients.Elements {
				key := client.DeviceId + "|" + client.DeviceName
				if ap, exists := apCounts[key]; exists {
					ap.Count++
					apCounts[key] = ap
				} else {
					apCounts[key] = struct {
						DeviceId   string
						DeviceName string
						Count      int
					}{
						DeviceId:   client.DeviceId,
						DeviceName: client.DeviceName,
						Count:      1,
					}
				}
			}

			// Reset all AP client counts to 0 first (for APs with no clients)
			inventory, err := c.client.GetInventory(site.ID)
			if err == nil {
				for _, device := range inventory.Elements {
					if device.DeviceType == "accessPoint" {
						clientsByAP.WithLabelValues(site.ID, site.Name, device.ID, device.Name).Set(0)
					}
				}
			}

			// Set actual client counts for APs that have clients
			for _, ap := range apCounts {
				clientsByAP.WithLabelValues(site.ID, site.Name, ap.DeviceId, ap.DeviceName).Set(float64(ap.Count))
			}
		}

		// LandingPage gives us throughput, 24h data volume, wired client count
		// (replacing the now-404 wiredClientSummary endpoint), and device up/down.
		if lp, err := c.client.GetLandingPage(site.ID); err != nil {
			log.Printf("Failed to get landing page for site %s: %v", site.Name, err)
		} else {
			labels := []string{site.ID, site.Name}
			wiredClientsTotal.WithLabelValues(labels...).Set(float64(lp.WiredClientsCount))
			wirelessClientsTotal.WithLabelValues(labels...).Set(float64(lp.WirelessClientsCount))
			siteThroughputBps.WithLabelValues(labels...).Set(float64(lp.CurrentNetworkThroughputInBitsPerSecond))
			siteData24hBytes.WithLabelValues(labels...).Set(float64(lp.TotalDataTransferredDuringLast24HoursInBytes))
			siteDevicesUp.WithLabelValues(labels...).Set(float64(lp.DeviceUpCount))
			siteNetworksConfigured.WithLabelValues(site.ID, site.Name, "total").Set(float64(lp.ConfiguredNetworksCount))
			siteNetworksConfigured.WithLabelValues(site.ID, site.Name, "wired").Set(float64(lp.ConfiguredWiredNetworksCount))
			siteNetworksConfigured.WithLabelValues(site.ID, site.Name, "wireless").Set(float64(lp.ConfiguredWirelessNetworksCount))
			siteNetworksActive.WithLabelValues(site.ID, site.Name, "total").Set(float64(lp.CurrentlyActiveNetworksCount))
			siteNetworksActive.WithLabelValues(site.ID, site.Name, "wired").Set(float64(lp.CurrentlyActiveWiredNetworksCount))
			siteNetworksActive.WithLabelValues(site.ID, site.Name, "wireless").Set(float64(lp.CurrentlyActiveWirelessNetworksCount))
		}

		if as, err := c.client.GetAlertsSummary(site.ID); err != nil {
			log.Printf("Failed to get alerts summary for site %s: %v", site.Name, err)
		} else {
			siteActiveAlerts.WithLabelValues(site.ID, site.Name, "info").Set(float64(as.ActiveInfoAlertsCount))
			siteActiveAlerts.WithLabelValues(site.ID, site.Name, "minor").Set(float64(as.ActiveMinorAlertsCount))
			siteActiveAlerts.WithLabelValues(site.ID, site.Name, "major").Set(float64(as.ActiveMajorAlertsCount))
		}
	}
}

func main() {
	fmt.Println("Starting Aruba Instant On Exporter...")

	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	username := os.Getenv("ARUBA_USERNAME")
	password := os.Getenv("ARUBA_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("ARUBA_USERNAME and ARUBA_PASSWORD environment variables are required")
	}

	client := NewArubaClient(username, password)

	// Test authentication and API
	log.Println("Testing authentication...")
	sites, err := client.GetSites()
	if err != nil {
		log.Printf("Failed to fetch sites: %v", err)
	} else {
		log.Printf("Authentication successful! Found %d sites", sites.TotalCount)
		for _, site := range sites.Elements {
			log.Printf("  - %s (%s): %s [%s]", site.Name, site.ID, site.Health, site.Status)
		}
	}

	// PROBE mode: hit a list of undocumented endpoints once against the first
	// site, dump the raw JSON, and exit. Used to discover response shapes for
	// metrics that aren't implemented yet.
	if os.Getenv("PROBE") == "1" {
		if sites == nil || len(sites.Elements) == 0 {
			log.Fatal("PROBE mode requires at least one site")
		}
		site := sites.Elements[0]
		probeEndpoints := []string{
			"/sites/" + site.ID + "/landingPage",
			"/sites/" + site.ID + "/alertsSummary",
			"/sites/" + site.ID + "/alerts",
			"/sites/" + site.ID + "/applicationCategoryUsage",
			"/sites/" + site.ID + "/networksSummary",
			"/sites/" + site.ID + "/wiredNetworks",
			"/sites/" + site.ID + "/wiredClientSummary",
			"/sites/" + site.ID + "/capabilities",
			"/sites/" + site.ID + "/clientBlacklist",
		}
		fmt.Printf("\n=== PROBE MODE: site %s (%s) ===\n\n", site.Name, site.ID)
		for _, ep := range probeEndpoints {
			status, body, err := client.ProbeEndpoint(ep)
			fmt.Printf("--- %s ---\n", ep)
			if err != nil {
				fmt.Printf("ERROR: %v\n\n", err)
				continue
			}
			fmt.Printf("HTTP %d\n%s\n\n", status, body)
		}
		return
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(sitesTotal)
	reg.MustRegister(siteInfo)
	reg.MustRegister(devicesTotal)
	reg.MustRegister(deviceInfo)
	reg.MustRegister(deviceUptime)
	reg.MustRegister(wirelessClientsTotal)
	reg.MustRegister(wiredClientsTotal)
	reg.MustRegister(clientsByNetwork)
	reg.MustRegister(clientsByAP)
	reg.MustRegister(siteThroughputBps)
	reg.MustRegister(siteData24hBytes)
	reg.MustRegister(siteDevicesUp)
	reg.MustRegister(siteActiveAlerts)
	reg.MustRegister(siteNetworksConfigured)
	reg.MustRegister(siteNetworksActive)

	collector := NewCollector(client)

	// Update metrics periodically
	go func() {
		for {
			collector.Collect()
			time.Sleep(30 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	port := ":9100"
	log.Printf("Server listening on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
