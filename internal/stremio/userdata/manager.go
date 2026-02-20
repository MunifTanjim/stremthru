package stremio_userdata

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/google/uuid"
)

var ErrUnsupportedUserdataFormat = errors.New("unsupported userdata format")

type Manager[T any] interface {
	Delete(ud UserData[T]) error
	Export(ud UserData[T]) (string, error)
	GetId(ud UserData[T]) string
	Import(encoded string, ud UserData[T]) error
	IsSaved(ud UserData[T]) bool
	Load(id string, ud UserData[T]) error
	Resolve(ud UserData[T]) error
	Save(ud UserData[T], name string) error
	Sync(ud UserData[T]) error
}

type ManagerConfig struct {
	AddonName string
}

type iManager[T any] struct {
	addon string
	cache cache.Cache[StremioUserData[T]]
}

func (m iManager[T]) encode(ud UserData[T]) error {
	blob, err := json.Marshal(ud)
	if err != nil {
		return err
	}
	ud.SetEncoded(util.Base64Encode(string(blob)))
	return nil
}

func (m iManager[T]) decode(ud UserData[T]) error {
	encoded := ud.GetEncoded()
	blob, err := util.Base64DecodeToByte(encoded)
	if err != nil {
		return err
	}
	err = json.Unmarshal(blob, ud.Ptr())
	if err != nil {
		return err
	}
	ud.SetEncoded(encoded)
	return nil
}

func (m iManager[T]) Load(id string, ud UserData[T]) (err error) {
	sud := &StremioUserData[T]{}
	if !m.cache.Get(id, sud) {
		sud, err = Get[T](m.addon, id)
		if err != nil {
			return err
		}
		if sud == nil {
			return shared.ErrorBadRequest(nil, "invalid userdata id")
		}
	}

	*ud.Ptr() = sud.Value
	ud.SetEncoded("k." + id)

	return m.cache.Add(id, *sud)
}

func (m iManager[T]) Resolve(ud UserData[T]) (err error) {
	encoded := ud.GetEncoded()

	if encoded == "" {
		return nil
	}

	if !strings.HasPrefix(encoded, "k.") {
		if config.Stremio.Locked {
			return ErrUnsupportedUserdataFormat
		}
		return m.decode(ud)
	}

	id := strings.TrimPrefix(encoded, "k.")
	return m.Load(id, ud)
}

func (m iManager[T]) IsSaved(ud UserData[T]) bool {
	return strings.HasPrefix(ud.GetEncoded(), "k.")
}

func (m iManager[T]) GetId(ud UserData[T]) string {
	if m.IsSaved(ud) {
		return strings.TrimPrefix(ud.GetEncoded(), "k.")
	}
	return ""
}

func (m iManager[T]) Save(ud UserData[T], name string) error {
	if m.IsSaved(ud) {
		return nil
	}

	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	ud.SetEncoded("k." + id)

	return Create(m.addon, id, name, ud)
}

func (m iManager[T]) Delete(ud UserData[T]) error {
	if !m.IsSaved(ud) {
		return nil
	}

	id := m.GetId(ud)
	err := Delete(m.addon, id)
	if err != nil {
		return err
	}
	m.cache.Remove(id)
	return m.encode(ud)
}

func (m iManager[T]) Sync(ud UserData[T]) error {
	if !m.IsSaved(ud) {
		return m.encode(ud)
	}

	id := m.GetId(ud)
	if err := Update(m.addon, id, ud); err != nil {
		return err
	}
	m.cache.Remove(id)
	worker_queue.LinkedUserdataAddonReloaderQueue.Queue(worker_queue.UserdataAddonReloaderQueueItem{
		Addon: m.addon,
		Key:   id,
	})
	return nil
}

func (m iManager[T]) Export(ud UserData[T]) (string, error) {
	blob, err := json.Marshal(ud.StripSecrets())
	if err != nil {
		return "", err
	}
	return util.Base64Encode(string(blob)), nil
}

func (m iManager[T]) Import(encodedBlob string, ud UserData[T]) error {
	blob, err := util.Base64Decode(encodedBlob)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(blob), ud)
}

func NewManager[T any](config *ManagerConfig) Manager[T] {
	manager := iManager[T]{
		addon: config.AddonName,
	}
	manager.cache = cache.NewCache[StremioUserData[T]](&cache.CacheConfig{
		Lifetime: 1 * time.Hour,
		Name:     "stremio.addon.userdata." + config.AddonName,
	})
	return manager
}
