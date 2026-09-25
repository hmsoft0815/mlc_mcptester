package client

import (
	"context"
	"testing"
	"time"
)

func TestNotificationsWait(t *testing.T) {
	n := &Notifications{}
	ctx := context.Background()

	// Received before the wait still counts, and is consumed
	n.add(Notification{Method: ToolsListChanged})
	if err := n.Wait(ctx, ToolsListChanged, "", 50*time.Millisecond); err != nil {
		t.Fatalf("Wait missed an earlier notification: %v", err)
	}
	if err := n.Wait(ctx, ToolsListChanged, "", 50*time.Millisecond); err == nil {
		t.Fatal("Wait returned a consumed notification twice")
	}

	// URI must match for resources/updated
	go func() {
		time.Sleep(30 * time.Millisecond)
		n.add(Notification{Method: ResourceUpdated, URI: "mcp://a"})
	}()
	if err := n.Wait(ctx, ResourceUpdated, "mcp://b", 100*time.Millisecond); err == nil {
		t.Fatal("Wait matched a different URI")
	}
	if err := n.Wait(ctx, ResourceUpdated, "mcp://a", 100*time.Millisecond); err != nil {
		t.Fatalf("Wait missed the matching URI: %v", err)
	}

	if err := n.Wait(ctx, "notifications/unknown", "", time.Millisecond); err == nil {
		t.Fatal("Wait accepted an unknown method")
	}
}
