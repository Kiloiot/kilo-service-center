// Package scaci provides opId sign classification per SCACI §3.2
package scaci

// CommandInitiator defines who can initiate a command per SCACI §3.2
type CommandInitiator string

const (
	// InitiatorAC indicates command is AC-initiated (positive opId required)
	InitiatorAC CommandInitiator = "ac"
	// InitiatorSC indicates command is SC-initiated (negative opId required)
	InitiatorSC CommandInitiator = "sc"
	// InitiatorEither indicates command can be initiated by either party (any sign valid)
	InitiatorEither CommandInitiator = "either"
	// InitiatorConnect indicates Connect handshake (opId=0 only, validated separately)
	InitiatorConnect CommandInitiator = "connect"
)
