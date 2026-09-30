/**
 * Endpoint form values shared by the create and edit dialogs, and the
 * create/update requests built from them (validation: endpoint-validation.ts).
 */

import type {
  CreateEndpointRequest,
  EndpointUI,
  UpdateEndpointRequest,
} from "@api-types/api";

import { formatEui, normalizeEui } from "@utils/eui";
import { formatShortAddress } from "@utils/formatters";
import { ENDPOINT_CARRIER_OFFSET_NONE } from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";

export interface EndpointFormValues {
  epEui: string;
  name: string;
  shortAddr: string;
  bidirectional: boolean;
  preAttach: boolean;
  carrierOffset: string;
  networkKey: string;
  applicationKey: string;
  /** Remove the stored application key on save, unless a new key is entered. */
  removeApplicationKey: boolean;
  dualChan: boolean;
  repetition: boolean;
  wideCarrOff: boolean;
  longBlkDist: boolean;
  lastPacketCnt: string;
  attachCnt: string;
  typeEui: string;
  deviceModelId: string;
}

export const EMPTY_ENDPOINT_FORM: EndpointFormValues = {
  epEui: "",
  name: "",
  shortAddr: "",
  bidirectional: false,
  preAttach: false,
  carrierOffset: "",
  networkKey: "",
  applicationKey: "",
  removeApplicationKey: false,
  dualChan: false,
  repetition: false,
  wideCarrOff: false,
  longBlkDist: false,
  lastPacketCnt: "",
  attachCnt: "",
  typeEui: "",
  deviceModelId: "",
};

/** Builds initial EndpointFormValues from a loaded EndpointUI. */
export function buildEditEndpointFormData(
  endpoint: EndpointUI,
): EndpointFormValues {
  return {
    epEui: endpoint.epEui ? formatEui(endpoint.epEui) : "",
    name: endpoint.name || "",
    shortAddr: endpoint.shAddr ? formatShortAddress(endpoint.shAddr) : "",
    bidirectional: endpoint.bidi ?? false,
    preAttach: endpoint.preAttach ?? false,
    carrierOffset:
      endpoint.carrierOffset !== undefined
        ? String(endpoint.carrierOffset)
        : "",
    networkKey: "",
    applicationKey: "",
    removeApplicationKey: false,
    dualChan: endpoint.dualChan ?? false,
    repetition: endpoint.repetition ?? false,
    wideCarrOff: endpoint.wideCarrOff ?? false,
    longBlkDist: endpoint.longBlkDist ?? false,
    lastPacketCnt:
      endpoint.lastPacketCnt !== undefined
        ? String(endpoint.lastPacketCnt)
        : "0",
    attachCnt:
      endpoint.attachCnt !== undefined ? String(endpoint.attachCnt) : "0",
    typeEui: endpoint.typeEui || "",
    deviceModelId: endpoint.deviceModelId || "",
  };
}

/** Apply epEui (cascade rename via newEpEui) + name diffs. */
export function applyEndpointIdentityChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  const eui = normalizeEui(form.epEui);
  if (eui !== normalizeEui(original.epEui) && eui !== "") {
    changes.newEpEui = eui;
  }
  if (form.name !== original.name) changes.name = form.name;
}

/** The carrier offset a field holds; an empty field records none. */
function carrierOffsetValue(field: string): number {
  return field === "" ? ENDPOINT_CARRIER_OFFSET_NONE : Number(field);
}

/** Apply shortAddr (hex) + carrierOffset diffs; an emptied offset is an explicit clear. */
export function applyEndpointAddressChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  if (form.shortAddr !== original.shortAddr && form.shortAddr !== "") {
    changes.shAddr = parseInt(form.shortAddr, 16);
  }
  if (form.carrierOffset !== original.carrierOffset) {
    changes.carrierOffset = carrierOffsetValue(form.carrierOffset);
  }
}

/** Apply networkKey (nwkSnKey) + applicationKey (appKey) diffs. */
export function applyEndpointSecurityChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  if (form.networkKey !== original.networkKey && form.networkKey !== "") {
    changes.nwkSnKey = form.networkKey;
  }
  if (
    form.applicationKey !== original.applicationKey &&
    form.applicationKey !== ""
  ) {
    changes.appKey = form.applicationKey;
  } else if (form.removeApplicationKey) {
    changes.appKey = null;
  }
}

/** Apply MIOTY radio-option diffs (bidi/preAttach/dualChan/repetition/wide/long). */
export function applyEndpointMiotyOptionChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  if (form.bidirectional !== original.bidirectional) {
    changes.bidi = form.bidirectional;
  }
  if (form.preAttach !== original.preAttach) {
    changes.preAttach = form.preAttach;
  }
  if (form.dualChan !== original.dualChan) {
    changes.dualChan = form.dualChan;
  }
  if (form.repetition !== original.repetition) {
    changes.repetition = form.repetition;
  }
  if (form.wideCarrOff !== original.wideCarrOff) {
    changes.wideCarrOff = form.wideCarrOff;
  }
  if (form.longBlkDist !== original.longBlkDist) {
    changes.longBlkDist = form.longBlkDist;
  }
}

/** Apply lastPacketCnt + attachCnt counter diffs. */
export function applyEndpointCounterChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  if (
    form.lastPacketCnt !== original.lastPacketCnt &&
    form.lastPacketCnt !== ""
  ) {
    changes.lastPacketCnt = parseInt(form.lastPacketCnt, 10);
  }
  if (form.attachCnt !== original.attachCnt && form.attachCnt !== "") {
    changes.attachCnt = parseInt(form.attachCnt, 10);
  }
}

/**
 * Apply typeEui + deviceModelId blueprint diffs. Empty strings on these
 * fields are surfaced as explicit `null` so the backend clears the column.
 */
export function applyEndpointBlueprintChanges(
  changes: UpdateEndpointRequest,
  form: EndpointFormValues,
  original: EndpointFormValues,
): void {
  if (form.typeEui !== original.typeEui) {
    changes.typeEui = form.typeEui || null;
  }
  if (form.deviceModelId !== original.deviceModelId) {
    changes.deviceModelId = form.deviceModelId || null;
  }
}

/**
 * Computes the diff between form data and the original snapshot, returning
 * an UpdateEndpointRequest with only the changed fields. EUI renames are
 * surfaced via newEpEui per the cascade rename contract; clearable string
 * fields use null to request explicit clear.
 */
export function buildChangedEndpointRequest(
  formData: EndpointFormValues,
  originalData: EndpointFormValues,
): UpdateEndpointRequest {
  const changes: UpdateEndpointRequest = {};
  applyEndpointIdentityChanges(changes, formData, originalData);
  applyEndpointAddressChanges(changes, formData, originalData);
  applyEndpointSecurityChanges(changes, formData, originalData);
  applyEndpointMiotyOptionChanges(changes, formData, originalData);
  applyEndpointCounterChanges(changes, formData, originalData);
  applyEndpointBlueprintChanges(changes, formData, originalData);
  return changes;
}

/** The form fields base stations hold: the attach propagate parameters (BSSCI §3.8.1). */
const STATION_PROFILE_FIELDS: ReadonlyArray<{
  field: keyof EndpointFormValues;
  label: string;
}> = [
  { field: "bidirectional", label: ENDPOINT_FORM.LABEL_BIDIRECTIONAL },
  { field: "networkKey", label: ENDPOINT_FORM.LABEL_NETWORK_KEY },
  { field: "shortAddr", label: ENDPOINT_FORM.LABEL_SHORT_ADDR },
  { field: "lastPacketCnt", label: ENDPOINT_FORM.LABEL_LAST_PACKET_CNT },
  { field: "dualChan", label: ENDPOINT_FORM.LABEL_DUAL_CHAN },
  { field: "repetition", label: ENDPOINT_FORM.LABEL_REPETITION },
  { field: "wideCarrOff", label: ENDPOINT_FORM.LABEL_WIDE_CARR_OFF },
  { field: "longBlkDist", label: ENDPOINT_FORM.LABEL_LONG_BLK_DIST },
];

/** Labels of the station profile fields an edit changed, in §3.8.1 order. */
export function changedStationProfileFields(
  formData: EndpointFormValues,
  originalData: EndpointFormValues,
): string[] {
  return STATION_PROFILE_FIELDS.filter(
    ({ field }) => formData[field] !== originalData[field],
  ).map(({ label }) => label);
}

/**
 * Builds the CreateEndpointRequest from validated form data. All MIOTY
 * protocol fields are required (BSSCI §3.8.1 / SCACI §3.6.1) and surfaced
 * with explicit boolean values for radio flags; optional fields are
 * omitted when blank.
 */
export function buildCreateEndpointPayload(
  formData: EndpointFormValues,
): CreateEndpointRequest {
  return {
    epEui: normalizeEui(formData.epEui),
    name: formData.name,
    shAddr: parseInt(formData.shortAddr, 16),
    nwkSnKey: formData.networkKey,
    bidi: formData.bidirectional,
    preAttach: formData.preAttach,
    dualChan: formData.dualChan,
    repetition: formData.repetition,
    wideCarrOff: formData.wideCarrOff,
    longBlkDist: formData.longBlkDist,
    lastPacketCnt: parseInt(formData.lastPacketCnt, 10),
    attachCnt: parseInt(formData.attachCnt, 10),
    appKey: formData.applicationKey || undefined,
    typeEui: formData.typeEui || undefined,
    carrierOffset: carrierOffsetValue(formData.carrierOffset),
    deviceModelId: formData.deviceModelId || undefined,
  };
}
