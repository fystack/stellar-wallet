package cluster

import (
	"net/http"
	"testing"
	"time"
)

func TestNodesFallsBackToThreeLocalNodes(t *testing.T) {
	client := NewClient(&http.Client{Timeout: 50 * time.Millisecond}, "127.0.0.1:1", 1)

	nodes := client.Nodes()
	if len(nodes) != 3 {
		t.Fatalf("len(Nodes()) = %d, want 3", len(nodes))
	}
	for index, node := range nodes {
		if node.Name != "node"+string(rune('0'+index)) || node.Region != "local" || node.Online {
			t.Fatalf("node[%d] = %#v", index, node)
		}
	}
}
