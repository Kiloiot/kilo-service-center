//go:build integration

package probe

// Spec vocabulary of the conformance suite. Every name below is copied from
// the MIOTY specifications, not from the Service Center's own catalogs, so a
// wrong product constant cannot make a conformance test agree with it.

// BSSCI §3.3-3.17 and SCACI §3.3-3.14 command names.
const (
	cmdCon            = "con"
	cmdConRsp         = "conRsp"
	cmdConCmp         = "conCmp"
	cmdPing           = "ping"
	cmdPingRsp        = "pingRsp"
	cmdPingCmp        = "pingCmp"
	cmdStatus         = "status"
	cmdStatusCmp      = "statusCmp"
	cmdAtt            = "att"
	cmdAttRsp         = "attRsp"
	cmdDet            = "det"
	cmdDetRsp         = "detRsp"
	cmdAttPrp         = "attPrp"
	cmdAttPrpCmp      = "attPrpCmp"
	cmdDetPrp         = "detPrp"
	cmdDetPrpCmp      = "detPrpCmp"
	cmdULData         = "ulData"
	cmdULDataRsp      = "ulDataRsp"
	cmdULDataTx       = "ulDataTx"
	cmdULDataTxCmp    = "ulDataTxCmp"
	cmdDLDataQue      = "dlDataQue"
	cmdDLDataQueCmp   = "dlDataQueCmp"
	cmdDLDataRev      = "dlDataRev"
	cmdDLDataRevCmp   = "dlDataRevCmp"
	cmdDLDataRes      = "dlDataRes"
	cmdDLDataResRsp   = "dlDataResRsp"
	cmdDLRxStat       = "dlRxStat"
	cmdDLRxStatRsp    = "dlRxStatRsp"
	cmdDLRxStatQry    = "dlRxStatQry"
	cmdDLRxStatQryCmp = "dlRxStatQryCmp"
	cmdError          = "error"
	cmdErrorAck       = "errorAck"
	cmdReg            = "reg"
	cmdDereg          = "dereg"
	cmdEPStat         = "epStat"
	cmdTxDataResRsp   = "txDataResRsp"
	cmdTxDataResCmp   = "txDataResCmp"
)

// Operation suffixes: every operation X is answered by XRsp and completed by
// XCmp (BSSCI §3.3-3.16, SCACI §3.3-3.13).
const (
	suffixRsp = "Rsp"
	suffixCmp = "Cmp"
)

// Message field names (BSSCI §3.3-3.17, SCACI §3.3-3.14).
const (
	keyCommand       = "command"
	keyOpID          = "opId"
	keyVersion       = "version"
	keyBsEui         = "bsEui"
	keyAcEui         = "acEui"
	keyScEui         = "scEui"
	keyVendor        = "vendor"
	keyModel         = "model"
	keyName          = "name"
	keySwVersion     = "swVersion"
	keyInfo          = "info"
	keyBidi          = "bidi"
	keySnBsUUID      = "snBsUuid"
	keySnAcUUID      = "snAcUuid"
	keySnScOpID      = "snScOpId"
	keySnAcOpID      = "snAcOpId"
	keySnResume      = "snResume"
	keySnScUUID      = "snScUuid"
	keyCode          = "code"
	keyMessage       = "message"
	keyTime          = "time"
	keyDutyCycle     = "dutyCycle"
	keyEpEui         = "epEui"
	keyRxTime        = "rxTime"
	keyRxDuration    = "rxDuration"
	keyAttachCnt     = "attachCnt"
	keySnr           = "snr"
	keyRssi          = "rssi"
	keyEqSnr         = "eqSnr"
	keyProfile       = "profile"
	keyMode          = "mode"
	keySubpackets    = "subpackets"
	keyFrequency     = "frequency"
	keyNonce         = "nonce"
	keySign          = "sign"
	keyShAddr        = "shAddr"
	keyDualChan      = "dualChan"
	keyRepetition    = "repetition"
	keyWideCarrOff   = "wideCarrOff"
	keyLongBlkDist   = "longBlkDist"
	keyNwkSnKey      = "nwkSnKey"
	keyNwkKey        = "nwkKey"
	keyPacketCnt     = "packetCnt"
	keyLastPacketCnt = "lastPacketCnt"
	keyUserData      = "userData"
	keyFormat        = "format"
	keyDlOpen        = "dlOpen"
	keyResponseExp   = "responseExp"
	keyDlAck         = "dlAck"
	keyQueID         = "queId"
	keyCntDepend     = "cntDepend"
	keyPrio          = "prio"
	keyResponsePrio  = "responsePrio"
	keyDlWindReq     = "dlWindReq"
	keyExpOnly       = "expOnly"
	keyDlRxStatQry   = "dlRxStatQry"
	keyResult        = "result"
	keyTxTime        = "txTime"
	keyDlRxSnr       = "dlRxSnr"
	keyDlRxRssi      = "dlRxRssi"
	keyBaseStations  = "baseStations"
	keyDuplicate     = "duplicate"
	keyPreAttach     = "preAttach"
	keyEPStatus      = "epStatus"
)

// Enumerated field values (BSSCI §3.14.1, SCACI §3.12.1 and §3.13.1).
const (
	resultSent       = "sent"
	resultExpired    = "expired"
	resultInvalid    = "invalid"
	epStatusAttached = "attached"
	epStatusDetached = "detached"
)

// POSIX error numbers the specifications use for error codes (BSSCI §3.17,
// SCACI §3.14).
const (
	posixEINVAL = 22
	posixERANGE = 34
	posixEPROTO = 71
)

// Numeric limits the specifications define.
const (
	protocolVersion   = "1.0.0" // BSSCI §2, SCACI §2
	maxRadioPayload   = 200     // RADIO §3.6.5.5 (UL) and §3.6.6.3 (DL)
	sessionKeyLen     = 16      // Numeric[16] keys and session UUIDs
	nonceLen          = 4       // Numeric[4] nonce and sign
	attachIVLen       = 16      // RADIO Fig. 3-15
	attachIVMarkerHi  = 0xFF    // RADIO Fig. 3-15, byte 8
	attachIVMarkerLo  = 0x00    // RADIO Fig. 3-15, byte 9
	attachIVTrailer   = 0xFFFF  // RADIO Fig. 3-15, bytes 14-15
	maxFormat         = 0xFF    // 8-bit format identifier
	firstSignedInt32  = 1 << 31 // smallest counter that needs the 32nd bit
	uint64AboveInt64  = 1<<63 + 5
	oversizedShAddr   = 70000 // does not fit the 16-bit short address
	oversizedFormat   = 300   // does not fit the 8-bit format identifier
	oversizedRadioLen = maxRadioPayload + 1
)

// bssciSCFields lists, per command the Service Center sends to a base
// station, every field the BSSCI tables define for it (§3.3-3.17). A1 checks
// SC frames against this list.
var bssciSCFields = map[string][]string{
	cmdConRsp: {keyCommand, keyOpID, keyVersion, keyScEui, keyVendor, keyModel, keyName, keySwVersion,
		keyInfo, keySnResume, keySnScUUID},
	cmdPing:           {keyCommand, keyOpID},
	cmdPingRsp:        {keyCommand, keyOpID},
	cmdPingCmp:        {keyCommand, keyOpID},
	cmdStatus:         {keyCommand, keyOpID},
	cmdStatusCmp:      {keyCommand, keyOpID},
	cmdAttRsp:         {keyCommand, keyOpID, keyNwkSnKey, keyShAddr},
	cmdDetRsp:         {keyCommand, keyOpID, keySign},
	cmdAttPrp:         {keyCommand, keyOpID, keyEpEui, keyBidi, keyNwkSnKey, keyShAddr, keyLastPacketCnt, keyDualChan, keyRepetition, keyWideCarrOff, keyLongBlkDist},
	cmdAttPrpCmp:      {keyCommand, keyOpID},
	cmdDetPrp:         {keyCommand, keyOpID, keyEpEui},
	cmdDetPrpCmp:      {keyCommand, keyOpID},
	cmdULDataRsp:      {keyCommand, keyOpID},
	cmdULDataTx:       {keyCommand, keyOpID, keyEpEui, keyNwkSnKey, keyShAddr, keyPacketCnt, keyProfile, keyUserData, keyFormat},
	cmdULDataTxCmp:    {keyCommand, keyOpID},
	cmdDLDataQue:      {keyCommand, keyOpID, keyEpEui, keyQueID, keyCntDepend, keyPacketCnt, keyUserData, keyFormat, keyPrio, keyResponseExp, keyResponsePrio, keyDlWindReq, keyExpOnly},
	cmdDLDataQueCmp:   {keyCommand, keyOpID},
	cmdDLDataRev:      {keyCommand, keyOpID, keyEpEui, keyQueID},
	cmdDLDataRevCmp:   {keyCommand, keyOpID},
	cmdDLDataResRsp:   {keyCommand, keyOpID},
	cmdDLRxStatRsp:    {keyCommand, keyOpID},
	cmdDLRxStatQry:    {keyCommand, keyOpID, keyEpEui},
	cmdDLRxStatQryCmp: {keyCommand, keyOpID},
	cmdError:          {keyCommand, keyOpID, keyCode, keyMessage},
	cmdErrorAck:       {keyCommand, keyOpID},
}

// scaciULDataFields and scaciULDataStationFields list every field SCACI §3.8.1
// defines for ulData and for one entry of its baseStations array.
var (
	scaciULDataFields = []string{keyCommand, keyOpID, keyEpEui, keyBaseStations, keyPacketCnt, keyUserData,
		keyFormat, keyDlOpen, keyResponseExp, keyDlAck, keyDuplicate}
	scaciULDataStationFields = []string{keyBsEui, keyRxTime, keyRxDuration, keySnr, keyRssi, keyEqSnr,
		keyDlRxSnr, keyDlRxRssi, keyProfile, keyMode, keySubpackets}
)
