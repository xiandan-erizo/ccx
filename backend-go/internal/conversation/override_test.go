package conversation

import (
	"testing"
	"time"
)

func TestOverrideManager_SetAndGet(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{
		{ChannelIndex: 1, ChannelName: "backup"},
		{ChannelIndex: 0, ChannelName: "primary"},
	}

	err := om.SetOverride("conv_abc", "chat", "user1", seq)
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}

	result, ok := om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Fatal("expected override to exist")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].ChannelIndex != 1 {
		t.Errorf("expected first channel index=1, got %d", result[0].ChannelIndex)
	}
}

func TestOverrideManager_Remove(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq)

	removed := om.RemoveOverride("conv_abc")
	if !removed {
		t.Error("expected RemoveOverride to return true")
	}

	_, ok := om.GetOverrideForUser("chat", "user1")
	if ok {
		t.Error("expected override to be removed")
	}
}

func TestOverrideManager_TTLExpiry(t *testing.T) {
	om := NewOverrideManager(1 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq)

	time.Sleep(5 * time.Millisecond)

	_, ok := om.GetOverrideForUser("chat", "user1")
	if ok {
		t.Error("expected override to be expired")
	}
}

func TestOverrideManager_EmptySequence(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	err := om.SetOverride("conv_abc", "chat", "user1", []ChannelEntry{})
	if err == nil {
		t.Error("expected error for empty sequence")
	}
}

func TestOverrideManager_GetAllOverrides(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	om.SetOverride("conv_1", "chat", "user1", []ChannelEntry{{ChannelIndex: 0, ChannelName: "a"}})
	om.SetOverride("conv_2", "messages", "user2", []ChannelEntry{{ChannelIndex: 1, ChannelName: "b"}})

	all := om.GetAllOverrides()
	if len(all) != 2 {
		t.Errorf("expected 2 overrides, got %d", len(all))
	}
}

func TestOverrideManager_RefreshTTL(t *testing.T) {
	om := NewOverrideManager(100 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq)

	time.Sleep(50 * time.Millisecond)
	om.RefreshTTL("conv_abc")
	time.Sleep(70 * time.Millisecond)

	_, ok := om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Error("expected override to still be valid after TTL refresh")
	}
}
