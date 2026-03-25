package server

import (
	"sync"
	"time"
)

var (
	maintenanceMu          sync.Mutex
	maintenanceEndsAt      time.Time
	maintenanceMaxDuration = 1 * time.Minute
)

func ActivateMaintenance(ttl time.Duration) {
	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()
	now := time.Now()
	endsAt := now.Add(ttl)
	if maxEndsAt := now.Add(maintenanceMaxDuration); endsAt.After(maxEndsAt) {
		endsAt = maxEndsAt
	}
	if endsAt.After(maintenanceEndsAt) {
		maintenanceEndsAt = endsAt.Truncate(time.Second)
	}
}

func DeactivateMaintenance() {
	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()
	maintenanceEndsAt = time.Time{}
}

func GetMaintenanceEndTime() time.Time {
	maintenanceMu.Lock()
	defer maintenanceMu.Unlock()
	return maintenanceEndsAt
}

func IsMaintenanceActive() bool {
	return time.Now().Before(GetMaintenanceEndTime())
}
