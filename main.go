package main

import (
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
		apiVersion: "7",
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

type WiredClient struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MacAddress    string `json:"macAddress"`
	ClientType    string `json:"clientType"`
	IsVoiceDevice bool   `json:"isVoiceDevice"`
	IPAddress     string `json:"ipAddress"`
}

type ClientSummaryResponse struct {
	TotalCount int              `json:"totalCount"`
	Elements   []WirelessClient `json:"elements"`
}

type WiredClientSummaryResponse struct {
	TotalCount int           `json:"totalCount"`
	Elements   []WiredClient `json:"elements"`
}

type SitesResponse struct {
	TotalCount int    `json:"totalCount"`
	Elements   []Site `json:"elements"`
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

func (c *ArubaClient) GetWiredClientSummary(siteID string) (*WiredClientSummaryResponse, error) {
	resp, err := c.Request("GET", "/sites/"+siteID+"/wiredClientSummary", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get wired client summary: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var wiredClientResp WiredClientSummaryResponse
	if err := json.Unmarshal(body, &wiredClientResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &wiredClientResp, nil
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

		// Get wireless clients for this site
		wirelessClients, err := c.client.GetClientSummary(site.ID)
		if err != nil {
			log.Printf("Failed to get wireless clients for site %s: %v", site.Name, err)
		} else {
			wirelessClientsTotal.WithLabelValues(site.ID, site.Name).Set(float64(wirelessClients.TotalCount))

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

		// Get wired clients for this site (temporarily disabled due to 404 error)
		// TODO: Fix wired client endpoint
		/*
			wiredClients, err := c.client.GetWiredClientSummary(site.ID)
			if err != nil {
				log.Printf("Failed to get wired clients for site %s: %v", site.Name, err)
			} else {
				wiredClientsTotal.WithLabelValues(site.ID, site.Name).Set(float64(wiredClients.TotalCount))
			}
		*/
		// Set wired clients to 0 for now
		wiredClientsTotal.WithLabelValues(site.ID, site.Name).Set(0)
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
