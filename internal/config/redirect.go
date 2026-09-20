package config

import (
	"log"
	"math/rand/v2"
	"net/url"
	"strings"
)

type RedirectConfigMap map[string][]string

func parseRedirectConfig(configStr string) RedirectConfigMap {
	redirectConfigMap := make(RedirectConfigMap)
	for item := range strings.FieldsSeq(configStr) {
		key, valueList, ok := strings.Cut(item, ":")
		if !ok || key == "" || valueList == "" {
			log.Fatalf("invalid STREMTHRU__REDIRECT__ config item: <%s>", item)
		}
		if _, exists := redirectConfigMap[key]; exists {
			log.Fatalf("duplicate STREMTHRU__REDIRECT__ config key: <%s>", key)
		}
		urlList := make([]string, 0, 4)
		for rawUrl := range strings.SplitSeq(valueList, ",") {
			if rawUrl = strings.TrimSpace(rawUrl); rawUrl == "" {
				continue
			}
			if u, err := url.Parse(rawUrl); err != nil || u.Scheme == "" || u.Host == "" {
				log.Fatalf("invalid STREMTHRU__REDIRECT__ url for: <%s>", key)
			}
			urlList = append(urlList, rawUrl)
		}
		if len(urlList) == 0 {
			log.Fatalf("no redirect url configured for: <%s>", key)
		}
		redirectConfigMap[key] = urlList
	}
	return redirectConfigMap
}

var Redirect = func() RedirectConfigMap {
	return parseRedirectConfig(getEnv("STREMTHRU__REDIRECT__"))
}()

func (rcm RedirectConfigMap) GetRandom(key string) string {
	urlList, ok := rcm[key]
	if !ok || len(urlList) == 0 {
		return ""
	}
	return urlList[rand.IntN(len(urlList))]
}
