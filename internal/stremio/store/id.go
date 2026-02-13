package stremio_store

import (
	"errors"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/premiumize"
)

func isStoreId(id string) bool {
	return strings.HasPrefix(id, "st:store:")
}

func getCatalogId(storeCode string) string {
	return "st:store:" + storeCode
}

func getIdPrefix(storeCode string) string {
	return getCatalogId(storeCode) + ":"
}

func getStoreActionId(storeCode string) string {
	return getIdPrefix(storeCode) + "action"
}

func getStoreActionIdPrefix(storeCode string) string {
	return getStoreActionId(storeCode) + ":"
}

const WEBDL_META_ID_INDICATOR = "webdls"

func getWebDLsMetaId(storeCode string) string {
	return getIdPrefix(storeCode) + WEBDL_META_ID_INDICATOR
}

func getWebDLsMetaIdPrefix(storeCode string) string {
	return getWebDLsMetaId(storeCode) + ":"
}

type ParsedId struct {
	storeCode    store.StoreCode
	storeName    store.StoreName
	isUsenet     bool
	isWebDL      bool
	isDeprecated bool
	isST         bool
	code         string
}

func (idr *ParsedId) getStoreCode() string {
	if idr.code == "" {
		if idr.isST {
			if idr.storeCode == "" {
				idr.code = "st"
			} else if idr.storeCode == "st" {
				if idr.isUsenet {
					idr.code = string(idr.storeCode) + "-usenet"
				}
			} else if idr.isDeprecated {
				idr.code = "st:" + string(idr.storeCode)
			} else {
				idr.code = "st-" + string(idr.storeCode)
				if idr.isUsenet {
					idr.code += "-usenet"
				} else if idr.isWebDL {
					idr.code += "-webdl"
				}
			}
		} else {
			idr.code = string(idr.storeCode)
			if idr.isUsenet {
				idr.code += "-usenet"
			} else if idr.isWebDL {
				idr.code += "-webdl"
			}
		}
	}
	return idr.code
}

func parseId(id string) (*ParsedId, error) {
	parts := strings.SplitN(id, ":", 5)
	count := len(parts)
	if count < 3 {
		return nil, errors.New("invalid id")
	}

	r := ParsedId{}
	storeCode := parts[2]
	if strings.Contains(storeCode, "-") {
		scParts := strings.Split(storeCode, "-")
		if scParts[0] == "st" {
			r.isST = true
			switch scParts[1] {
			case "usenet":
				r.storeCode = store.StoreCodeStremThru
				r.isUsenet = true
			default:
				r.storeCode = store.StoreCode(scParts[1])
			}
			if len(scParts) > 2 {
				switch scParts[2] {
				case "usenet":
					r.isUsenet = true
				case "webdl":
					r.isWebDL = true
				}
			}
		} else {
			r.storeCode = store.StoreCode(scParts[0])
			if len(scParts) > 1 {
				switch scParts[1] {
				case "usenet":
					r.isUsenet = true
				case "webdl":
					r.isWebDL = true
				}
			}
		}
	} else {
		switch storeCode {
		case "st":
			r.isST = true
			r.isDeprecated = true
			if count > 3 {
				r.storeCode = store.StoreCode(parts[3])
				if r.storeCode.Name() == "" {
					r.storeCode = ""
				}
			}
		default:
			r.storeCode = store.StoreCode(parts[2])
		}
	}

	r.storeName = r.storeCode.Name()

	return &r, nil
}

func getVideoIdAndData(idWithLink string, idr *ParsedId) (id, link, name string, err error) {
	videoId := strings.TrimPrefix(idWithLink, getIdPrefix(idr.getStoreCode()))
	idPrefix := ""
	switch idr.storeCode {
	case store.StoreCodePremiumize:
		if strings.HasPrefix(videoId, premiumize.CachedMagnetIdPrefix) {
			idPrefix = premiumize.CachedMagnetIdPrefix
			videoId = strings.TrimPrefix(videoId, premiumize.CachedMagnetIdPrefix)
		}
	}
	id, escapedLink, _ := strings.Cut(videoId, ":")
	link, err = url.PathUnescape(escapedLink)
	if err != nil {
		return "", "", "", err
	}
	if l, err := url.Parse(link); err != nil || l.Scheme == "" {
		name = link
		link = ""
	}
	return idPrefix + id, link, name, nil
}
