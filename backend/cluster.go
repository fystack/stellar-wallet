package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// nodeHealthReady pings a node's /health endpoint (local nodes: 8091, 8092, ...).
func nodeHealthReady(index int) bool {
	basePort, _ := strconv.Atoi(getenv("HEALTH_BASE_PORT", "8091"))
	url := fmt.Sprintf("http://localhost:%d/health", basePort+index)
	resp, err := httpc.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// "live" = this node's process is up; "ready" is cluster-wide readiness.
	var h struct {
		Live bool `json:"live"`
	}
	if json.NewDecoder(resp.Body).Decode(&h) != nil {
		return false
	}
	return h.Live
}

type ClusterNode struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Online bool   `json:"online"`
}

type kvEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"` // base64
}

func consulKV(prefix string) ([]kvEntry, error) {
	base := getenv("CONSUL_ADDR", "10.10.0.1:8500")
	resp, err := httpc.Get("http://" + base + "/v1/kv/" + prefix + "?recurse")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil
	}
	var out []kvEntry
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func decode(v string) string {
	b, _ := base64.StdEncoding.DecodeString(v)
	return string(b)
}

// clusterNodes reads the real peer list + readiness flags from Consul.
// Falls back to a static 3-node view if Consul is unreachable.
func clusterNodes() []ClusterNode {
	peers, err := consulKV("mpc_peers/")
	if err != nil || len(peers) == 0 {
		// Fall back to pinging the local health ports directly.
		nodes := []ClusterNode{}
		for i := 0; i < 3; i++ {
			nodes = append(nodes, ClusterNode{
				Name:   fmt.Sprintf("node%d", i),
				Region: "local",
				Online: nodeHealthReady(i),
			})
		}
		return nodes
	}

	names := make([]string, 0, len(peers))
	for _, p := range peers {
		names = append(names, strings.TrimPrefix(p.Key, "mpc_peers/"))
	}
	sort.Strings(names)

	nodes := make([]ClusterNode, 0, len(names))
	for i, name := range names {
		// Node index parsed from "nodeN" so the health port matches.
		idx := i
		if n, err := strconv.Atoi(strings.TrimPrefix(name, "node")); err == nil {
			idx = n
		}
		nodes = append(nodes, ClusterNode{
			Name:   name,
			Region: "local",
			Online: nodeHealthReady(idx),
		})
	}
	return nodes
}
