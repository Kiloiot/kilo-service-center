package mioty

// ReceptionAt is the reception of the message by the base station, one of
// the SCACI §3.8.1 baseStations; ok is false when that station did not
// receive it.
func (m *ULDataMessage) ReceptionAt(bsEui uint64) (*BaseStationReception, bool) {
	for i := range m.BaseStations {
		if m.BaseStations[i].BsEui == bsEui {
			return &m.BaseStations[i], true
		}
	}
	return nil, false
}
