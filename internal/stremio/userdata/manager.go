package stremio_userdata

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/google/uuid"
)

type Manager[T any] interface {
	Delete(ud UserData[T]) error
	GetId(ud UserData[T]) string
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
	ud.SetEncoded(core.Base64Encode(string(blob)))
	return nil
}

func (m iManager[T]) decode(ud UserData[T]) error {
	encoded := ud.GetEncoded()
	blob, err := core.Base64DecodeToByte(encoded)
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
			if err == sql.ErrNoRows {
				e := core.NewAPIError("invalid userdata id")
				e.Code = core.ErrorCodeBadRequest
				e.Cause = err
				return e
			}
			return err
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
	return nil
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
