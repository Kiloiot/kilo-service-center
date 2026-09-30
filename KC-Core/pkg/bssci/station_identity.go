package bssci

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StationDirectory finds a tenant's registered base station by EUI.
type StationDirectory interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
}

// StationIdentity is how an event names a base station: always its EUI, and
// its name and ID only when it is registered to the tenant the event is filed
// under, so a roaming endpoint's owner never learns another tenant's names.
type StationIdentity struct {
	EUI  string
	Name string
	ID   *int64
}

// IdentifyStation names the base station bsEUI for an event of tenantID.
func IdentifyStation(ctx context.Context, stations StationDirectory, tenantID int64, bsEUI uint64) StationIdentity {
	identity := StationIdentity{EUI: mioty.FormatEUI64(bsEUI)}
	eui := mioty.EUI64(bsEUI).ToBytes()
	station, err := stations.GetByEUI(ctx, tenantID, eui[:])
	if err != nil || station == nil {
		return identity
	}
	identity.Name = station.Name
	identity.ID = &station.ID
	return identity
}

// Label renders the station as "name (EUI)", or its EUI when it has no name.
func (s StationIdentity) Label() string {
	if s.Name == "" {
		return s.EUI
	}
	return fmt.Sprintf(stationLabelFmt, s.Name, s.EUI)
}
