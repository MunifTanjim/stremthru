package config

import (
	"log"
	"slices"
	"strings"
)

const (
	FeatureMeta string = "meta"
	FeatureNewz string = "newz"
	FeatureTorz string = "torz"
	FeatureSync string = "sync"

	FeatureAnime           string = "anime"
	FeaturePerformance     string = "performance"
	FeatureDMMHashlist     string = "dmm_hashlist"
	FeatureIMDBTitle       string = "imdb_title"
	FeatureStremioList     string = "stremio_list"
	FeatureStremioNewz     string = "stremio_newz"
	FeatureStremioP2P      string = "stremio_p2p"
	FeatureStremioSidekick string = "stremio_sidekick"
	FeatureStremioStore    string = "stremio_store"
	FeatureStremioTorz     string = "stremio_torz"
	FeatureStremioWrap     string = "stremio_wrap"
	FeatureVault           string = "vault"
	FeatureProbeMediaInfo  string = "probe_media_info"
)

var features = []string{
	FeatureMeta,
	FeatureNewz,
	FeatureTorz,
	FeatureSync,
	FeatureVault,

	FeatureAnime,
	FeaturePerformance,
	FeatureDMMHashlist,
	FeatureIMDBTitle,

	FeatureStremioList,
	FeatureStremioNewz,
	FeatureStremioP2P,
	FeatureStremioSidekick,
	FeatureStremioStore,
	FeatureStremioTorz,
	FeatureStremioWrap,

	FeatureProbeMediaInfo,
}

type FeatureConfig struct {
	enabled  []string
	disabled []string
}

func (f FeatureConfig) IsDisabled(name string) bool {
	return slices.Contains(f.disabled, name)
}

func (f FeatureConfig) IsEnabled(name string) bool {
	if f.IsDisabled(name) {
		return false
	}

	if len(f.enabled) == 0 {
		return true
	}

	return slices.Contains(f.enabled, name)
}

func (f FeatureConfig) HasMeta() bool {
	return f.IsEnabled(FeatureMeta)
}

func (f FeatureConfig) HasNewz() bool {
	return f.IsEnabled(FeatureNewz)
}

func (f FeatureConfig) HasTorz() bool {
	return f.IsEnabled(FeatureTorz)
}

// Performance requires MatchLab's precomputed file identities. Falling back to
// the original path without MatchLab also preserves original result semantics.
func (f FeatureConfig) CanUsePerformance(hasMatchLab bool) bool {
	return hasMatchLab && f.HasTorz() && f.IsEnabled(FeaturePerformance)
}

func (f FeatureConfig) HasPerformance() bool {
	return f.CanUsePerformance(Integration.MatchLab.IsEnabled())
}

func (f FeatureConfig) HasSync() bool {
	return f.IsEnabled(FeatureSync) && f.HasVault()
}

func (f FeatureConfig) HasVault() bool {
	return !f.IsDisabled(FeatureVault) && VaultSecret != ""
}

func (f FeatureConfig) HasDMMHashlist() bool {
	return f.IsEnabled(FeatureDMMHashlist) && f.HasTorz()
}

func (f FeatureConfig) HasIMDBTitle() bool {
	return f.IsEnabled(FeatureIMDBTitle) && (f.HasNewz() || f.HasTorz())
}

func (f FeatureConfig) HasStremioList() bool {
	return f.IsEnabled(FeatureStremioList)
}

func (f FeatureConfig) HasStremioNewz() bool {
	return f.IsEnabled(FeatureStremioNewz) && f.HasNewz()
}

func (f FeatureConfig) HasStremioTorz() bool {
	return f.IsEnabled(FeatureStremioTorz) && f.HasTorz()
}

func (f FeatureConfig) HasStremioStore() bool {
	return f.IsEnabled(FeatureStremioStore) && (f.HasNewz() || f.HasTorz())
}

func (f FeatureConfig) HasStremioAddon() bool {
	return f.HasStremioList() || f.HasStremioNewz() || f.HasStremioTorz() || f.HasStremioStore() || f.IsEnabled(FeatureStremioWrap)
}

func (f FeatureConfig) HasProbeMediaInfo() bool {
	return f.IsEnabled(FeatureProbeMediaInfo) && f.HasTorz()
}

func parseFeatureConfig(value string) FeatureConfig {
	feature := FeatureConfig{
		disabled: []string{FeatureAnime, FeaturePerformance, FeatureStremioP2P},
	}

	for _, name := range strings.FieldsFunc(strings.TrimSpace(value), func(c rune) bool {
		return c == ','
	}) {
		switch {
		case strings.HasPrefix(name, "-"):
			name = strings.TrimPrefix(name, "-")
			if slices.Contains(feature.enabled, name) {
				log.Fatalf("feature conflict, trying to disable already enabled feature: -%s", name)
			} else {
				feature.disabled = append(feature.disabled, name)
			}
		case strings.HasPrefix(name, "+"):
			name = strings.TrimPrefix(name, "+")
			if slices.Contains(feature.disabled, name) {
				feature.disabled = slices.DeleteFunc(feature.disabled, func(feat string) bool {
					return feat == name
				})
			} else {
				log.Fatalf("feature conflict, trying to force enable a not disabled feature: +%s", name)
			}
		default:
			if slices.Contains(feature.disabled, name) {
				log.Fatalf("feature conflict, trying to enable already disabled feature: %s", name)
			} else {
				feature.enabled = append(feature.enabled, name)
			}
		}
	}

	return feature
}

var Feature = parseFeatureConfig(getEnv("STREMTHRU_FEATURE"))
