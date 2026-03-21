package campaign

import (
	"testing"
)

func TestMemoryCampaignStore_SaveAndGet(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{
		Name:        "test",
		Description: "desc",
		Status:      StatusDraft,
		CreatedBy:   "cli",
	}
	if err := store.SaveCampaign(c); err != nil {
		t.Fatalf("SaveCampaign: %v", err)
	}
	if c.CampaignID == "" {
		t.Error("expected CampaignID to be set")
	}
	got, err := store.GetCampaign(c.CampaignID)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	if got.Name != c.Name || got.CampaignID != c.CampaignID {
		t.Errorf("got %+v, want %+v", got, c)
	}
}

func TestMemoryCampaignStore_ListCampaigns(t *testing.T) {
	store := NewMemoryCampaignStore()
	c1 := &Campaign{Name: "a", Status: StatusDraft, CreatedBy: "cli"}
	c2 := &Campaign{Name: "b", Status: StatusDraft, CreatedBy: "cli"}
	_ = store.SaveCampaign(c1)
	_ = store.SaveCampaign(c2)
	list, err := store.ListCampaigns()
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d campaigns, want 2", len(list))
	}
}

func TestMemoryCampaignStore_UpdateCampaign(t *testing.T) {
	store := NewMemoryCampaignStore()
	c := &Campaign{Name: "x", Status: StatusDraft, CreatedBy: "cli"}
	_ = store.SaveCampaign(c)
	c.Description = "updated"
	if err := store.UpdateCampaign(c); err != nil {
		t.Fatalf("UpdateCampaign: %v", err)
	}
	got, _ := store.GetCampaign(c.CampaignID)
	if got.Description != "updated" {
		t.Errorf("got description %q, want updated", got.Description)
	}
}

func TestMemoryCampaignStore_GetNotFound(t *testing.T) {
	store := NewMemoryCampaignStore()
	_, err := store.GetCampaign("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent campaign")
	}
}
