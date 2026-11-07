package config

import "testing"

func TestAnimeAndPerformanceFeatureModes(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		wantAnime bool
		wantFast  bool
		wantTorz  bool
	}{
		{name: "defaults", env: "", wantTorz: true},
		{name: "anime only", env: "+anime", wantAnime: true, wantTorz: true},
		{name: "performance only", env: "+performance", wantFast: true, wantTorz: true},
		{name: "both features", env: "+anime,+performance", wantAnime: true, wantFast: true, wantTorz: true},
		{name: "explicitly disable anime", env: "+performance,-anime", wantFast: true, wantTorz: true},
		{name: "explicitly disable performance", env: "+anime,-performance", wantAnime: true, wantTorz: true},
		{name: "whitelist torz only", env: "torz", wantTorz: true},
		{name: "performance without torz", env: "+performance,-torz", wantFast: false, wantTorz: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := parseFeatureConfig(tc.env)
			if got := f.IsEnabled(FeatureAnime); got != tc.wantAnime {
				t.Errorf("anime enabled = %t, want %t", got, tc.wantAnime)
			}
			if got := f.CanUsePerformance(true); got != tc.wantFast {
				t.Errorf("performance enabled = %t, want %t", got, tc.wantFast)
			}
			if got := f.HasTorz(); got != tc.wantTorz {
				t.Errorf("torz enabled = %t, want %t", got, tc.wantTorz)
			}
			if f.CanUsePerformance(false) {
				t.Error("performance must fall back to original behavior when MatchLab is disabled")
			}
		})
	}
}
