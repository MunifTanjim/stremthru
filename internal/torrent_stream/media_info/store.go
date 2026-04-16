package media_info

import "errors"

type StoreProber func(apiKey, linkId string) (*MediaInfo, error)

var storeProberByCode = map[string]StoreProber{}

func RegisterStoreProber(storeCode string, prober StoreProber) {
	storeProberByCode[storeCode] = prober
}

func ProbeStore(storeCode string, storeToken string, linkId string) (*MediaInfo, error) {
	prober, ok := storeProberByCode[storeCode]
	if !ok {
		return nil, errors.New("unsupported store code")
	}
	mi, err := prober(storeToken, linkId)
	if err != nil {
		return nil, err
	}
	return mi, nil
}
