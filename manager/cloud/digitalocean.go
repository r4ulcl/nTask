package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

func ProcessDigitalOcean(configCloud *utils.ManagerCloudConfig, configSSH *utils.ManagerSSHConfig, verbose, debug bool) {
	doClient := &DigitalOceanClient{Token: configCloud.ApiKey}

	// Check snapshot exists
	snapshot, err := doClient.GetSnapshotByName(context.Background(), configCloud.SnapshotName)
	if err != nil {
		log.Fatal("Error GetSnapshotByName:", err)
	}

	if configCloud.Recreate {
		// Delete all droplets by prefix
		if debug {
			log.Println("Delete all droplet with prefix:", configCloud.SnapshotName)
		}
		err = doClient.DeleteDropletsByPrefix(context.Background(), configCloud.SnapshotName)
		if err != nil {
			log.Fatal("Error GetSnapshotByName:", err)
		}
	}

	// Get current number of droplets
	if debug {
		log.Println("List droplets by prefix")
	}
	droplets, err := doClient.ListDropletsByPrefix(context.Background(), configCloud.SnapshotName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	numDroplets := len(droplets)
	if numDroplets < configCloud.Servers {
		missingDroplets := configCloud.Servers - numDroplets
		if debug {
			log.Println("Creating multiple droplets from snapshot")
		}
		// Create new droplets
		ids, err := doClient.CreateXDropletsFromSnapshot(context.Background(), configCloud.SnapshotName, snapshot.ID, configCloud.Region, configCloud.Size, configCloud.SshKeys, missingDroplets, numDroplets)
		if err != nil {
			log.Println("Error CreateXDropletsFromSnapshot:", err)
		}

		// Wait until all has IP
		for _, id := range ids {
			if debug {
				log.Println("Waiting for:", id)
			}
			ip, err := doClient.WaitForDropletCreation(context.Background(), id, verbose, debug)
			if err != nil {
				log.Println("Error WaitForDropletCreation:", err)
			}
			if debug {
				log.Println("Droplet with ID:", id, " IP: ", ip)
			}
		}
	}
	// Get all IPs and add to array
	ips, err := doClient.GetDropletIPsByPrefix(context.Background(), configCloud.SnapshotName)
	if err != nil {
		fmt.Println("Error:", err)
	}

	for _, ip := range ips {
		log.Println(ip)
		configSSH.IPPort[ip] = fmt.Sprint(configCloud.SshPort)
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

// ListDroplets retrieves all droplets associated with the account
func (c *DigitalOceanClient) ListDroplets(ctx context.Context) ([]Droplet, error) {
	req, err := http.NewRequest("GET", digitalOceanBaseURL+"/droplets", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var droplets struct {
		Droplets []Droplet `json:"droplets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&droplets); err != nil {
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

// ListSnapshots retrieves all snapshots associated with the account
func (c *DigitalOceanClient) ListSnapshots(ctx context.Context) ([]Snapshot, error) {
	req, err := http.NewRequest("GET", digitalOceanBaseURL+"/snapshots", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var snapshots struct {
		Snapshots []Snapshot `json:"snapshots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		return nil, err
	}

	return snapshots.Snapshots, nil
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

// --------------
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

type DropletResponse struct {
	ID int `json:"id"`
}

type DigitalOceanResponse struct {
	Droplets []DropletResponse `json:"droplets"`
}

func (c *DigitalOceanClient) CreateXDropletsFromSnapshot(ctx context.Context, name, snapshotID, region, size, sshKey string, count, startNumber int) ([]int, error) {
	// Create the payload for the request
	names := make([]string, count)

	for i := 0; i < count; i++ {
		names[i] = name + "-" + strconv.Itoa(i+1+startNumber)
	}

	payload := RequestPayload{
		Names:      names,
		Region:     region,
		Size:       size,
		Image:      snapshotID,
		SSHKeys:    []interface{}{sshKey},
		Backups:    false,
		IPv6:       false,
		Monitoring: false,
		Tags:       []string{"nTask", "r4ulcl"},
	}

	// Convert the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	log.Println(string(payloadBytes))

	// Send the request to create the droplets
	url := digitalOceanBaseURL + "/droplets"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Set your DigitalOcean API token here
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("failed to create droplets: %s", body)
	}

	// Parse the response
	var dropletsResponse DigitalOceanResponse
	err = json.Unmarshal(body, &dropletsResponse)
	if err != nil {
		return nil, err
	}

	// Collect the IDs of the new droplets
	ids := make([]int, len(dropletsResponse.Droplets))
	for i, droplet := range dropletsResponse.Droplets {
		ids[i] = droplet.ID
	}

	return ids, nil
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

// DeleteDroplet deletes a Droplet by its ID
func (c *DigitalOceanClient) DeleteDroplet(ctx context.Context, dropletID int) error {
	req, err := http.NewRequest("DELETE", digitalOceanBaseURL+"/droplets/"+fmt.Sprint(dropletID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
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
