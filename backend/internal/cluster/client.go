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
	healthURLTmpl string
	assumeOnline  bool
}

func NewClient(httpClient *http.Client, consulAddress string, healthBase int, healthURLTmpl string, assumeOnline bool) *Client {
	return &Client{
		httpClient:    httpClient,
		consulAddress: consulAddress,
		healthBase:    healthBase,
		healthURLTmpl: healthURLTmpl,
		assumeOnline:  assumeOnline,
	}
}

// healthURL builds a node's /health endpoint. With no template it targets
// localhost on a per-node port (health_base+index); with a template it fills
// {i} (index) and {port} so nodes can live on separate hosts (Docker).
func (c *Client) healthURL(index int) string {
	port := c.healthBase + index
	if c.healthURLTmpl == "" {
		return fmt.Sprintf("http://localhost:%d/health", port)
	}
	return strings.NewReplacer(
		"{i}", strconv.Itoa(index), "{port}", strconv.Itoa(port),
	).Replace(c.healthURLTmpl)
}

// OnlineCount reports how many MPC nodes are currently live. Used as a
// preflight before dispatching keygen (needs all 3) or signing (needs 2-of-3),
// so requests fail fast instead of hanging when nodes are down.
func (c *Client) OnlineCount() int {
	online := 0
	for _, node := range c.Nodes() {
		if node.Online {
			online++
		}
	}
	return online
}

func (c *Client) nodeReady(index int) bool {
	// No health endpoint (e.g. the official mpcium image) — a peer present in
	// Consul is taken as live; the timeout watchdog covers actual failures.
	if c.assumeOnline {
		return true
	}
	response, err := c.httpClient.Get(c.healthURL(index))
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
