package cluster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"stellar-wallet-backend/internal/domain"
)

type Client struct {
	httpClient    *http.Client
	consulAddress string
	healthBase    int
}

func NewClient(httpClient *http.Client, consulAddress string, healthBase int) *Client {
	return &Client{httpClient: httpClient, consulAddress: consulAddress, healthBase: healthBase}
}

func (c *Client) nodeReady(index int) bool {
	url := fmt.Sprintf("http://localhost:%d/health", c.healthBase+index)
	response, err := c.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	var health struct {
		Live bool `json:"live"`
	}
	return json.NewDecoder(response.Body).Decode(&health) == nil && health.Live
}

type kvEntry struct {
	Key string `json:"Key"`
}

func (c *Client) peers() ([]kvEntry, error) {
	response, err := c.httpClient.Get("http://" + c.consulAddress + "/v1/kv/mpc_peers/?recurse")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, nil
	}
	var peers []kvEntry
	if err := json.NewDecoder(response.Body).Decode(&peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func (c *Client) Nodes() []domain.ClusterNode {
	peers, err := c.peers()
	if err != nil || len(peers) == 0 {
		nodes := make([]domain.ClusterNode, 0, 3)
		for index := range 3 {
			nodes = append(nodes, domain.ClusterNode{
				Name: fmt.Sprintf("node%d", index), Region: "local", Online: c.nodeReady(index),
			})
		}
		return nodes
	}

	names := make([]string, 0, len(peers))
	for _, peer := range peers {
		names = append(names, strings.TrimPrefix(peer.Key, "mpc_peers/"))
	}
	sort.Strings(names)

	nodes := make([]domain.ClusterNode, 0, len(names))
	for fallbackIndex, name := range names {
		index := fallbackIndex
		if parsed, parseErr := strconv.Atoi(strings.TrimPrefix(name, "node")); parseErr == nil {
			index = parsed
		}
		nodes = append(nodes, domain.ClusterNode{Name: name, Region: "local", Online: c.nodeReady(index)})
	}
	return nodes
}
