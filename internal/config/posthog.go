package config

type posthogConfig struct {
	APIKey     string
	DistinctId string
}

func (conf posthogConfig) IsEnabled() bool {
	return conf.APIKey != ""
}

var Posthog = func() posthogConfig {
	conf := posthogConfig{}

	conf.APIKey = getEnv("POSTHOG_API_KEY")

	if conf.IsEnabled() {
		conf.DistinctId = BaseURL.Hostname()
	}

	return conf
}()
