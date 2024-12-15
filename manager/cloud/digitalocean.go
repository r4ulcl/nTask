// Package cloud to all the nTask manager cloud management
package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/r4ulcl/nTask/manager/utils"
)

const digitalOceanBaseURL = "https://api.digitalocean.com/v2"

// DigitalOceanClient represents the DigitalOcean API client
type DigitalOceanClient struct {
	Token string
}

// Droplet represents a DigitalOcean droplet
type Droplet struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Networks struct {
		V4 []struct {
			IPAddress string `json:"ip_address"`
		} `json:"v4"`
	} `json:"networks"`
	// Add other fields as needed
}

// ProcessDigitalOcean Process Digital Ocean config
func ProcessDigitalOcean(configCloud *utils.ManagerCloudConfig, configSSH *utils.ManagerSSHConfig, verbose, debug bool) {
	doClient := &DigitalOceanClient{Token: configCloud.APIKey}

	// Step 1: Check if snapshot exists
	snapshot, err := getSnapshotByName(doClient, configCloud.SnapshotName)
	if err != nil {
		log.Fatal("Error GetSnapshotByName:", err)
	}

	// Step 2: Recreate droplets if needed
	if configCloud.Recreate {
		deleteDropletsByPrefix(doClient, configCloud.SnapshotName, debug)
	}

	// Step 3: List current droplets and create new ones if necessary
	droplets, err := listDroplets(doClient, configCloud.SnapshotName, debug)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Step 4: Create missing droplets from snapshot if needed
	createMissingDroplets(doClient, configCloud, snapshot, droplets, debug, verbose)

	// Step 5: Get all IPs and add to SSH config
	updateSSHConfigWithIPs(doClient, configCloud.SnapshotName, configCloud.SSHPort, configSSH)
}

// Helper function: Get snapshot by name
func getSnapshotByName(doClient *DigitalOceanClient, snapshotName string) (*Snapshot, error) {
	return doClient.GetSnapshotByName(context.Background(), snapshotName)
}

// Helper function: Delete droplets by prefix
func deleteDropletsByPrefix(doClient *DigitalOceanClient, snapshotName string, debug bool) {
	if debug {
		log.Println("Delete all droplets with prefix:", snapshotName)
	}
	err := doClient.DeleteDropletsByPrefix(context.Background(), snapshotName)
	if err != nil {
		log.Fatal("Error DeleteDropletsByPrefix:", err)
	}
}

// Helper function: List droplets by prefix
func listDroplets(doClient *DigitalOceanClient, snapshotName string, debug bool) ([]Droplet, error) {
	if debug {
		log.Println("List droplets by prefix:", snapshotName)
	}
	return doClient.ListDropletsByPrefix(context.Background(), snapshotName)
}

// Helper function: Create missing droplets
func createMissingDroplets(doClient *DigitalOceanClient, configCloud *utils.ManagerCloudConfig, snapshot *Snapshot, droplets []Droplet, debug, verbose bool) {
	numDroplets := len(droplets)
	if numDroplets < configCloud.Servers {
		missingDroplets := configCloud.Servers - numDroplets
		if debug {
			log.Println("Creating multiple droplets from snapshot")
		}
		ids, err := doClient.CreateXDropletsFromSnapshot(context.Background(), configCloud.SnapshotName, snapshot.ID, configCloud.Region, configCloud.Size, configCloud.SSHKeys, missingDroplets, numDroplets)
		if err != nil {
			log.Println("Error CreateXDropletsFromSnapshot:", err)
		}

		// Wait until all droplets have an IP
		waitForDropletCreation(doClient, ids, verbose, debug)
	}
}

// Helper function: Wait for droplet creation and log IPs
func waitForDropletCreation(doClient *DigitalOceanClient, ids []int, verbose, debug bool) {
	for _, id := range ids {
		if debug || verbose {
			log.Println("Waiting for droplet:", id)
		}
		ip, err := doClient.WaitForDropletCreation(context.Background(), id, verbose, debug)
		if err != nil {
			log.Println("Error WaitForDropletCreation:", err)
		}
		if debug {
			log.Println("Droplet with ID:", id, " IP:", ip)
		}
	}
}

// Helper function: Get all IPs and update SSH config
func updateSSHConfigWithIPs(doClient *DigitalOceanClient, snapshotName string, sshPort int, configSSH *utils.ManagerSSHConfig) {
	ips, err := doClient.GetDropletIPsByPrefix(context.Background(), snapshotName)
	if err != nil {
		fmt.Println("Error:", err)
	}

	for _, ip := range ips {
		log.Println(ip)
		configSSH.IPPort[ip] = fmt.Sprint(sshPort)
	}
}

// ListDropletByID retrieves a droplet by ID
func (c *DigitalOceanClient) ListDropletByID(ctx context.Context, id int) (*Droplet, error) {
	droplets, err := c.ListDroplets(ctx)
	if err != nil {
		return nil, err
	}

	for _, droplet := range droplets {
		if droplet.ID == id {
			return &droplet, nil
		}
	}

	return nil, fmt.Errorf("Droplet with ID %d not found", id)
}

// ListSnapshots retrieves all snapshots associated with the account
func (c *DigitalOceanClient) ListSnapshots(ctx context.Context) ([]Snapshot, error) {
	var snapshots struct {
		Snapshots []Snapshot `json:"snapshots"`
	}

	err := c.fetchResources(ctx, "/snapshots", &snapshots)
	if err != nil {
		return nil, err
	}

	return snapshots.Snapshots, nil
}

// ListDroplets retrieves all droplets associated with the account
func (c *DigitalOceanClient) ListDroplets(ctx context.Context) ([]Droplet, error) {
	var droplets struct {
		Droplets []Droplet `json:"droplets"`
	}

	err := c.fetchResources(ctx, "/droplets", &droplets)
	if err != nil {
		return nil, err
	}

	return droplets.Droplets, nil
}

// ListDropletsByPrefix retrieves droplets filtered by name prefix
func (c *DigitalOceanClient) ListDropletsByPrefix(ctx context.Context, prefix string) ([]Droplet, error) {
	droplets, err := c.ListDroplets(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Droplet
	for _, droplet := range droplets {
		if len(droplet.Name) >= len(prefix) && droplet.Name[:len(prefix)] == prefix {
			filtered = append(filtered, droplet)
		}
	}

	return filtered, nil
}

// GetDropletIPsByPrefix retrieves IP addresses of droplets filtered by name prefix
func (c *DigitalOceanClient) GetDropletIPsByPrefix(ctx context.Context, prefix string) ([]string, error) {
	droplets, err := c.ListDropletsByPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, droplet := range droplets {
		for _, v4 := range droplet.Networks.V4 {
			if isPublicIP(v4.IPAddress) {
				ips = append(ips, v4.IPAddress)
			}
		}
	}

	return ips, nil
}

// isPublicIP checks if an IP address is public
func isPublicIP(ipAddress string) bool {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return false
	}

	// Private IP ranges as per RFC 1918
	privateRanges := []string{
		"10.",      // 10.0.0.0    - 10.255.255.255
		"172.16.",  // 172.16.0.0  - 172.31.255.255
		"192.168.", // 192.168.0.0 - 192.168.255.255
		"100.64.",  // 100.64.0.0  - 100.127.255.255 (Shared Address Space RFC 6598)
		"169.254.", // 169.254.0.0 - 169.254.255.255 (Link-local address)
	}

	for _, pr := range privateRanges {
		if strings.HasPrefix(ipAddress, pr) {
			return false
		}
	}

	return !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}

// Snapshot represents a DigitalOcean snapshot
type Snapshot struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetSnapshotByName retrieves a snapshot by its exact name
func (c *DigitalOceanClient) GetSnapshotByName(ctx context.Context, snapshotName string) (*Snapshot, error) {
	snapshots, err := c.ListSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		if snapshot.Name == snapshotName {
			return &snapshot, nil
		}
	}

	return nil, errors.New("snapshot not found")
}

// GetSnapshotsByPrefix retrieves snapshots filtered by name prefix
func (c *DigitalOceanClient) GetSnapshotsByPrefix(ctx context.Context, prefix string) ([]Snapshot, error) {
	snapshots, err := c.ListSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Snapshot
	for _, snapshot := range snapshots {
		if len(snapshot.Name) >= len(prefix) && snapshot.Name[:len(prefix)] == prefix {
			filtered = append(filtered, snapshot)
		}
	}

	return filtered, nil
}

// RequestPayload Request Payload for Digital Ocean
type RequestPayload struct {
	Names      []string      `json:"names"`
	Region     string        `json:"region"`
	Size       string        `json:"size"`
	Image      string        `json:"image"`
	SSHKeys    []interface{} `json:"ssh_keys"`
	Backups    bool          `json:"backups"`
	IPv6       bool          `json:"ipv6"`
	Monitoring bool          `json:"monitoring"`
	Tags       []string      `json:"tags"`
}

// DropletResponse Droplet Response
type DropletResponse struct {
	ID int `json:"id"`
}

// DigitalOceanResponse Digital Ocean Response
type DigitalOceanResponse struct {
	Droplets []DropletResponse `json:"droplets"`
}

// sendRequest is a reusable helper function to handle HTTP requests.
// It supports different HTTP methods and payloads, and parses the response.
func (c *DigitalOceanClient) sendRequest(ctx context.Context, method, endpoint string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, digitalOceanBaseURL+endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, response: %s", resp.Status, string(responseBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return err
		}
	}

	return nil
}

// CreateXDropletsFromSnapshot creates multiple droplets from a snapshot.
func (c *DigitalOceanClient) CreateXDropletsFromSnapshot(ctx context.Context, name, snapshotID, region, size, sshKey string, count, startNumber int) ([]int, error) {
	// Prepare the names for the droplets
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = name + "-" + strconv.Itoa(i+1+startNumber)
	}

	// Create the payload
	payload := RequestPayload{
		Names:      names,
		Region:     region,
		Size:       size,
		Image:      snapshotID,
		SSHKeys:    []interface{}{sshKey},
		Backups:    false,
		IPv6:       false,
		Monitoring: false,
		Tags:       []string{"nTask", "worker"},
	}

	// Response structure for droplet creation
	var dropletsResponse DigitalOceanResponse

	// Send POST request to create droplets
	if err := c.sendRequest(ctx, "POST", "/droplets", payload, &dropletsResponse); err != nil {
		return nil, err
	}

	// Collect and return the droplet IDs
	ids := make([]int, len(dropletsResponse.Droplets))
	for i, droplet := range dropletsResponse.Droplets {
		ids[i] = droplet.ID
	}

	return ids, nil
}

// DeleteDroplet deletes a droplet by its ID.
func (c *DigitalOceanClient) DeleteDroplet(ctx context.Context, dropletID int) error {
	// Send DELETE request to delete the droplet
	return c.sendRequest(ctx, "DELETE", fmt.Sprintf("/droplets/%d", dropletID), nil, nil)
}

// fetchResources fetches resources using a GET request and parses the response.
func (c *DigitalOceanClient) fetchResources(ctx context.Context, endpoint string, result interface{}) error {
	// Reuse sendRequest for GET requests
	return c.sendRequest(ctx, "GET", endpoint, nil, result)
}

//-----------------

// WaitForDropletCreation waits until a droplet with the given ID is created
// and returns its public IP address.
func (c *DigitalOceanClient) WaitForDropletCreation(ctx context.Context, dropletID int, verbose, debug bool) (string, error) {
	for {
		droplet, err := c.ListDropletByID(ctx, dropletID)
		if err != nil {
			if debug {
				log.Println("Droplet with ID ", dropletID, ":", err)
			}
		}

		if debug {
			log.Println("WaitForDropletCreation:", droplet)
		}

		if droplet != nil && droplet.Status == "active" {
			for _, network := range droplet.Networks.V4 {
				if network.IPAddress != "" {
					return network.IPAddress, nil
				}
			}
			return "", errors.New("unable to find public IP address")
		}

		select {
		case <-time.After(10 * time.Second):
			continue
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// DeleteDropletsByPrefix deletes all Droplets with the specified prefix
func (c *DigitalOceanClient) DeleteDropletsByPrefix(ctx context.Context, prefix string) error {
	droplets, err := c.ListDropletsByPrefix(ctx, prefix)
	if err != nil {
		return err
	}

	for _, droplet := range droplets {
		err := c.DeleteDroplet(ctx, droplet.ID)
		if err != nil {
			return fmt.Errorf("failed to delete droplet %d: %v", droplet.ID, err)
		}
	}

	// Introduce a short delay to wait for the deletion to complete
	time.Sleep(10 * time.Second)

	return nil
}
