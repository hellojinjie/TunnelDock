package ui

import "testing"

func TestDeleteConfirmationNamesTunnel(t *testing.T) {
	p := PresentDeleteConfirmation("Jupyter")
	if p.Title != "Delete Jupyter?" || !p.Destructive || p.PrimaryText != "Delete" {
		t.Fatalf("presentation = %#v", p)
	}
}

func TestSettingsPresentationExplainsSystemAppearance(t *testing.T) {
	p := PresentSettings(true, AppearanceDark)
	if !p.ShowTrayIcon || p.AppearanceText != "Appearance follows Windows (Dark)" {
		t.Fatalf("presentation = %#v", p)
	}
}
