// package: kilocenter.api.v1
// file: core.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as google_protobuf_field_mask_pb from "google-protobuf/google/protobuf/field_mask_pb";
import * as google_protobuf_struct_pb from "google-protobuf/google/protobuf/struct_pb";
import * as google_protobuf_wrappers_pb from "google-protobuf/google/protobuf/wrappers_pb";

export class EndPoint extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getEpClass(): string;
  setEpClass(value: string): void;

  getNwkSnKey(): Uint8Array | string;
  getNwkSnKey_asU8(): Uint8Array;
  getNwkSnKey_asB64(): string;
  setNwkSnKey(value: Uint8Array | string): void;

  getAppKey(): Uint8Array | string;
  getAppKey_asU8(): Uint8Array;
  getAppKey_asB64(): string;
  setAppKey(value: Uint8Array | string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getTagsMap(): jspb.Map<string, string>;
  clearTagsMap(): void;
  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastSeenAt(): boolean;
  clearLastSeenAt(): void;
  getLastSeenAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastSeenAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getAttachStatus(): string;
  setAttachStatus(value: string): void;

  getShAddr(): number;
  setShAddr(value: number): void;

  getDualChan(): boolean;
  setDualChan(value: boolean): void;

  getRepetition(): boolean;
  setRepetition(value: boolean): void;

  getWideCarrOff(): boolean;
  setWideCarrOff(value: boolean): void;

  getLongBlkDist(): boolean;
  setLongBlkDist(value: boolean): void;

  getAttachCnt(): number;
  setAttachCnt(value: number): void;

  getPreAttach(): boolean;
  setPreAttach(value: boolean): void;

  getLastPacketCnt(): number;
  setLastPacketCnt(value: number): void;

  getTypeEui(): Uint8Array | string;
  getTypeEui_asU8(): Uint8Array;
  getTypeEui_asB64(): string;
  setTypeEui(value: Uint8Array | string): void;

  getCarrierOffset(): number;
  setCarrierOffset(value: number): void;

  getDeviceModelId(): string;
  setDeviceModelId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EndPoint.AsObject;
  static toObject(includeInstance: boolean, msg: EndPoint): EndPoint.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EndPoint, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EndPoint;
  static deserializeBinaryFromReader(message: EndPoint, reader: jspb.BinaryReader): EndPoint;
}

export namespace EndPoint {
  export type AsObject = {
    epeui: string,
    tenantId: string,
    name: string,
    description: string,
    epClass: string,
    nwkSnKey: Uint8Array | string,
    appKey: Uint8Array | string,
    status: string,
    tagsMap: Array<[string, string]>,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastSeenAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    attachStatus: string,
    shAddr: number,
    dualChan: boolean,
    repetition: boolean,
    wideCarrOff: boolean,
    longBlkDist: boolean,
    attachCnt: number,
    preAttach: boolean,
    lastPacketCnt: number,
    typeEui: Uint8Array | string,
    carrierOffset: number,
    deviceModelId: string,
  }
}

export class BaseStation extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  hasLatitude(): boolean;
  clearLatitude(): void;
  getLatitude(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setLatitude(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasLongitude(): boolean;
  clearLongitude(): void;
  getLongitude(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setLongitude(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasAltitude(): boolean;
  clearAltitude(): void;
  getAltitude(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setAltitude(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  getStatus(): string;
  setStatus(value: string): void;

  getTagsMap(): jspb.Map<string, string>;
  clearTagsMap(): void;
  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastSeenAt(): boolean;
  clearLastSeenAt(): void;
  getLastSeenAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastSeenAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasSystemTime(): boolean;
  clearSystemTime(): void;
  getSystemTime(): google_protobuf_wrappers_pb.Int64Value | undefined;
  setSystemTime(value?: google_protobuf_wrappers_pb.Int64Value): void;

  hasDutyCycle(): boolean;
  clearDutyCycle(): void;
  getDutyCycle(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setDutyCycle(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasUptimeSeconds(): boolean;
  clearUptimeSeconds(): void;
  getUptimeSeconds(): google_protobuf_wrappers_pb.Int64Value | undefined;
  setUptimeSeconds(value?: google_protobuf_wrappers_pb.Int64Value): void;

  hasTemperatureCelsius(): boolean;
  clearTemperatureCelsius(): void;
  getTemperatureCelsius(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setTemperatureCelsius(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasCpuLoad(): boolean;
  clearCpuLoad(): void;
  getCpuLoad(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setCpuLoad(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasMemoryLoad(): boolean;
  clearMemoryLoad(): void;
  getMemoryLoad(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setMemoryLoad(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasBsConfig(): boolean;
  clearBsConfig(): void;
  getBsConfig(): google_protobuf_struct_pb.Struct | undefined;
  setBsConfig(value?: google_protobuf_struct_pb.Struct): void;

  hasLastStatusAt(): boolean;
  clearLastStatusAt(): void;
  getLastStatusAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastStatusAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getServiceCenterUrl(): string;
  setServiceCenterUrl(value: string): void;

  getLocationSource(): string;
  setLocationSource(value: string): void;

  hasLocationUpdatedAt(): boolean;
  clearLocationUpdatedAt(): void;
  getLocationUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLocationUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStation.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStation): BaseStation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStation;
  static deserializeBinaryFromReader(message: BaseStation, reader: jspb.BinaryReader): BaseStation;
}

export namespace BaseStation {
  export type AsObject = {
    bseui: string,
    tenantId: string,
    name: string,
    description: string,
    latitude?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    longitude?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    altitude?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    status: string,
    tagsMap: Array<[string, string]>,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastSeenAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    systemTime?: google_protobuf_wrappers_pb.Int64Value.AsObject,
    dutyCycle?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    uptimeSeconds?: google_protobuf_wrappers_pb.Int64Value.AsObject,
    temperatureCelsius?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    cpuLoad?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    memoryLoad?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    bsConfig?: google_protobuf_struct_pb.Struct.AsObject,
    lastStatusAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    serviceCenterUrl: string,
    locationSource: string,
    locationUpdatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class SubpacketInfo extends jspb.Message {
  clearSnrList(): void;
  getSnrList(): Array<number>;
  setSnrList(value: Array<number>): void;
  addSnr(value: number, index?: number): number;

  clearRssiList(): void;
  getRssiList(): Array<number>;
  setRssiList(value: Array<number>): void;
  addRssi(value: number, index?: number): number;

  clearFrequencyList(): void;
  getFrequencyList(): Array<number>;
  setFrequencyList(value: Array<number>): void;
  addFrequency(value: number, index?: number): number;

  clearPhaseList(): void;
  getPhaseList(): Array<number>;
  setPhaseList(value: Array<number>): void;
  addPhase(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubpacketInfo.AsObject;
  static toObject(includeInstance: boolean, msg: SubpacketInfo): SubpacketInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubpacketInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubpacketInfo;
  static deserializeBinaryFromReader(message: SubpacketInfo, reader: jspb.BinaryReader): SubpacketInfo;
}

export namespace SubpacketInfo {
  export type AsObject = {
    snrList: Array<number>,
    rssiList: Array<number>,
    frequencyList: Array<number>,
    phaseList: Array<number>,
  }
}

export class BaseStationReceptionInfo extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getRxTime(): number;
  setRxTime(value: number): void;

  getSnr(): number;
  setSnr(value: number): void;

  getRssi(): number;
  setRssi(value: number): void;

  hasEqSnr(): boolean;
  clearEqSnr(): void;
  getEqSnr(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setEqSnr(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasRxDuration(): boolean;
  clearRxDuration(): void;
  getRxDuration(): google_protobuf_wrappers_pb.Int64Value | undefined;
  setRxDuration(value?: google_protobuf_wrappers_pb.Int64Value): void;

  hasProfile(): boolean;
  clearProfile(): void;
  getProfile(): google_protobuf_wrappers_pb.StringValue | undefined;
  setProfile(value?: google_protobuf_wrappers_pb.StringValue): void;

  hasMode(): boolean;
  clearMode(): void;
  getMode(): google_protobuf_wrappers_pb.StringValue | undefined;
  setMode(value?: google_protobuf_wrappers_pb.StringValue): void;

  hasDlRxSnr(): boolean;
  clearDlRxSnr(): void;
  getDlRxSnr(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setDlRxSnr(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasDlRxRssi(): boolean;
  clearDlRxRssi(): void;
  getDlRxRssi(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setDlRxRssi(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  hasSubpackets(): boolean;
  clearSubpackets(): void;
  getSubpackets(): SubpacketInfo | undefined;
  setSubpackets(value?: SubpacketInfo): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationReceptionInfo.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationReceptionInfo): BaseStationReceptionInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationReceptionInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationReceptionInfo;
  static deserializeBinaryFromReader(message: BaseStationReceptionInfo, reader: jspb.BinaryReader): BaseStationReceptionInfo;
}

export namespace BaseStationReceptionInfo {
  export type AsObject = {
    bsEui: string,
    rxTime: number,
    snr: number,
    rssi: number,
    eqSnr?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    rxDuration?: google_protobuf_wrappers_pb.Int64Value.AsObject,
    profile?: google_protobuf_wrappers_pb.StringValue.AsObject,
    mode?: google_protobuf_wrappers_pb.StringValue.AsObject,
    dlRxSnr?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    dlRxRssi?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    subpackets?: SubpacketInfo.AsObject,
  }
}

export class Message extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEpeui(): string;
  setEpeui(value: string): void;

  getBseui(): string;
  setBseui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): void;

  getFrequency(): number;
  setFrequency(value: number): void;

  getRssi(): number;
  setRssi(value: number): void;

  getSnr(): number;
  setSnr(value: number): void;

  getEqSnr(): number;
  setEqSnr(value: number): void;

  getUplinkMode(): string;
  setUplinkMode(value: string): void;

  getPacketCounter(): number;
  setPacketCounter(value: number): void;

  getDlOpen(): boolean;
  setDlOpen(value: boolean): void;

  getResExp(): boolean;
  setResExp(value: boolean): void;

  getDlAck(): boolean;
  setDlAck(value: boolean): void;

  getCryptoMode(): number;
  setCryptoMode(value: number): void;

  hasReceivedAt(): boolean;
  clearReceivedAt(): void;
  getReceivedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setReceivedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDecodedPayload(): Uint8Array | string;
  getDecodedPayload_asU8(): Uint8Array;
  getDecodedPayload_asB64(): string;
  setDecodedPayload(value: Uint8Array | string): void;

  getDecodeStatus(): string;
  setDecodeStatus(value: string): void;

  getDecodeErrorCode(): string;
  setDecodeErrorCode(value: string): void;

  getBlueprintTypeEui(): Uint8Array | string;
  getBlueprintTypeEui_asU8(): Uint8Array;
  getBlueprintTypeEui_asB64(): string;
  setBlueprintTypeEui(value: Uint8Array | string): void;

  getBlueprintVersionId(): string;
  setBlueprintVersionId(value: string): void;

  clearBaseStationsList(): void;
  getBaseStationsList(): Array<BaseStationReceptionInfo>;
  setBaseStationsList(value: Array<BaseStationReceptionInfo>): void;
  addBaseStations(value?: BaseStationReceptionInfo, index?: number): BaseStationReceptionInfo;

  getDuplicate(): boolean;
  setDuplicate(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Message.AsObject;
  static toObject(includeInstance: boolean, msg: Message): Message.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Message, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Message;
  static deserializeBinaryFromReader(message: Message, reader: jspb.BinaryReader): Message;
}

export namespace Message {
  export type AsObject = {
    id: string,
    epeui: string,
    bseui: string,
    tenantId: string,
    payload: Uint8Array | string,
    frequency: number,
    rssi: number,
    snr: number,
    eqSnr: number,
    uplinkMode: string,
    packetCounter: number,
    dlOpen: boolean,
    resExp: boolean,
    dlAck: boolean,
    cryptoMode: number,
    receivedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    decodedPayload: Uint8Array | string,
    decodeStatus: string,
    decodeErrorCode: string,
    blueprintTypeEui: Uint8Array | string,
    blueprintVersionId: string,
    baseStationsList: Array<BaseStationReceptionInfo.AsObject>,
    duplicate: boolean,
  }
}

export class CreateEndPointRequest extends jspb.Message {
  hasEndpoint(): boolean;
  clearEndpoint(): void;
  getEndpoint(): EndPoint | undefined;
  setEndpoint(value?: EndPoint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateEndPointRequest): CreateEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateEndPointRequest;
  static deserializeBinaryFromReader(message: CreateEndPointRequest, reader: jspb.BinaryReader): CreateEndPointRequest;
}

export namespace CreateEndPointRequest {
  export type AsObject = {
    endpoint?: EndPoint.AsObject,
  }
}

export class GetEndPointRequest extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEndPointRequest): GetEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEndPointRequest;
  static deserializeBinaryFromReader(message: GetEndPointRequest, reader: jspb.BinaryReader): GetEndPointRequest;
}

export namespace GetEndPointRequest {
  export type AsObject = {
    epeui: string,
    tenantId: string,
  }
}

export class UpdateEndPointRequest extends jspb.Message {
  hasEndpoint(): boolean;
  clearEndpoint(): void;
  getEndpoint(): EndPoint | undefined;
  setEndpoint(value?: EndPoint): void;

  hasUpdateMask(): boolean;
  clearUpdateMask(): void;
  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): void;

  getNewEpEui(): string;
  setNewEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateEndPointRequest): UpdateEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateEndPointRequest;
  static deserializeBinaryFromReader(message: UpdateEndPointRequest, reader: jspb.BinaryReader): UpdateEndPointRequest;
}

export namespace UpdateEndPointRequest {
  export type AsObject = {
    endpoint?: EndPoint.AsObject,
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    newEpEui: string,
  }
}

export class DeleteEndPointRequest extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteEndPointRequest): DeleteEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteEndPointRequest;
  static deserializeBinaryFromReader(message: DeleteEndPointRequest, reader: jspb.BinaryReader): DeleteEndPointRequest;
}

export namespace DeleteEndPointRequest {
  export type AsObject = {
    epeui: string,
    tenantId: string,
  }
}

export class ListEndPointsRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getStatusFilter(): string;
  setStatusFilter(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndPointsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndPointsRequest): ListEndPointsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndPointsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndPointsRequest;
  static deserializeBinaryFromReader(message: ListEndPointsRequest, reader: jspb.BinaryReader): ListEndPointsRequest;
}

export namespace ListEndPointsRequest {
  export type AsObject = {
    tenantId: string,
    pageSize: number,
    pageToken: string,
    statusFilter: string,
  }
}

export class ListEndPointsResponse extends jspb.Message {
  clearEndpointsList(): void;
  getEndpointsList(): Array<EndPoint>;
  setEndpointsList(value: Array<EndPoint>): void;
  addEndpoints(value?: EndPoint, index?: number): EndPoint;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndPointsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndPointsResponse): ListEndPointsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndPointsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndPointsResponse;
  static deserializeBinaryFromReader(message: ListEndPointsResponse, reader: jspb.BinaryReader): ListEndPointsResponse;
}

export namespace ListEndPointsResponse {
  export type AsObject = {
    endpointsList: Array<EndPoint.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class AttachEndPointRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AttachEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AttachEndPointRequest): AttachEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AttachEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AttachEndPointRequest;
  static deserializeBinaryFromReader(message: AttachEndPointRequest, reader: jspb.BinaryReader): AttachEndPointRequest;
}

export namespace AttachEndPointRequest {
  export type AsObject = {
    epEui: string,
  }
}

export class AttachEndPointResponse extends jspb.Message {
  getOperationId(): string;
  setOperationId(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AttachEndPointResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AttachEndPointResponse): AttachEndPointResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AttachEndPointResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AttachEndPointResponse;
  static deserializeBinaryFromReader(message: AttachEndPointResponse, reader: jspb.BinaryReader): AttachEndPointResponse;
}

export namespace AttachEndPointResponse {
  export type AsObject = {
    operationId: string,
    status: string,
  }
}

export class DetachEndPointRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DetachEndPointRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DetachEndPointRequest): DetachEndPointRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DetachEndPointRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DetachEndPointRequest;
  static deserializeBinaryFromReader(message: DetachEndPointRequest, reader: jspb.BinaryReader): DetachEndPointRequest;
}

export namespace DetachEndPointRequest {
  export type AsObject = {
    epEui: string,
  }
}

export class DetachEndPointResponse extends jspb.Message {
  getOperationId(): string;
  setOperationId(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DetachEndPointResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DetachEndPointResponse): DetachEndPointResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DetachEndPointResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DetachEndPointResponse;
  static deserializeBinaryFromReader(message: DetachEndPointResponse, reader: jspb.BinaryReader): DetachEndPointResponse;
}

export namespace DetachEndPointResponse {
  export type AsObject = {
    operationId: string,
    status: string,
  }
}

export class CreateBaseStationRequest extends jspb.Message {
  hasBasestation(): boolean;
  clearBasestation(): void;
  getBasestation(): BaseStation | undefined;
  setBasestation(value?: BaseStation): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateBaseStationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateBaseStationRequest): CreateBaseStationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateBaseStationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateBaseStationRequest;
  static deserializeBinaryFromReader(message: CreateBaseStationRequest, reader: jspb.BinaryReader): CreateBaseStationRequest;
}

export namespace CreateBaseStationRequest {
  export type AsObject = {
    basestation?: BaseStation.AsObject,
  }
}

export class GetBaseStationRequest extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationRequest): GetBaseStationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationRequest;
  static deserializeBinaryFromReader(message: GetBaseStationRequest, reader: jspb.BinaryReader): GetBaseStationRequest;
}

export namespace GetBaseStationRequest {
  export type AsObject = {
    bseui: string,
    tenantId: string,
  }
}

export class UpdateBaseStationRequest extends jspb.Message {
  hasBasestation(): boolean;
  clearBasestation(): void;
  getBasestation(): BaseStation | undefined;
  setBasestation(value?: BaseStation): void;

  hasUpdateMask(): boolean;
  clearUpdateMask(): void;
  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBaseStationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBaseStationRequest): UpdateBaseStationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateBaseStationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBaseStationRequest;
  static deserializeBinaryFromReader(message: UpdateBaseStationRequest, reader: jspb.BinaryReader): UpdateBaseStationRequest;
}

export namespace UpdateBaseStationRequest {
  export type AsObject = {
    basestation?: BaseStation.AsObject,
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
  }
}

export class DeleteBaseStationRequest extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteBaseStationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteBaseStationRequest): DeleteBaseStationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteBaseStationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteBaseStationRequest;
  static deserializeBinaryFromReader(message: DeleteBaseStationRequest, reader: jspb.BinaryReader): DeleteBaseStationRequest;
}

export namespace DeleteBaseStationRequest {
  export type AsObject = {
    bseui: string,
    tenantId: string,
  }
}

export class UpdateBaseStationEuiRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getNewBsEui(): string;
  setNewBsEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBaseStationEuiRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBaseStationEuiRequest): UpdateBaseStationEuiRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateBaseStationEuiRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBaseStationEuiRequest;
  static deserializeBinaryFromReader(message: UpdateBaseStationEuiRequest, reader: jspb.BinaryReader): UpdateBaseStationEuiRequest;
}

export namespace UpdateBaseStationEuiRequest {
  export type AsObject = {
    bsEui: string,
    newBsEui: string,
  }
}

export class ListBaseStationsRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getStatusFilter(): string;
  setStatusFilter(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationsRequest): ListBaseStationsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationsRequest;
  static deserializeBinaryFromReader(message: ListBaseStationsRequest, reader: jspb.BinaryReader): ListBaseStationsRequest;
}

export namespace ListBaseStationsRequest {
  export type AsObject = {
    tenantId: string,
    pageSize: number,
    pageToken: string,
    statusFilter: string,
  }
}

export class ListBaseStationsResponse extends jspb.Message {
  clearBasestationsList(): void;
  getBasestationsList(): Array<BaseStation>;
  setBasestationsList(value: Array<BaseStation>): void;
  addBasestations(value?: BaseStation, index?: number): BaseStation;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationsResponse): ListBaseStationsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationsResponse;
  static deserializeBinaryFromReader(message: ListBaseStationsResponse, reader: jspb.BinaryReader): ListBaseStationsResponse;
}

export namespace ListBaseStationsResponse {
  export type AsObject = {
    basestationsList: Array<BaseStation.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class GetBaseStationStatsRequest extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationStatsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationStatsRequest): GetBaseStationStatsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationStatsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationStatsRequest;
  static deserializeBinaryFromReader(message: GetBaseStationStatsRequest, reader: jspb.BinaryReader): GetBaseStationStatsRequest;
}

export namespace GetBaseStationStatsRequest {
  export type AsObject = {
    bseui: string,
    tenantId: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetBaseStationStatsResponse extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  getTotalMessages(): number;
  setTotalMessages(value: number): void;

  getTotalEndpoints(): number;
  setTotalEndpoints(value: number): void;

  getMessagesToday(): number;
  setMessagesToday(value: number): void;

  getMessagesThisWeek(): number;
  setMessagesThisWeek(value: number): void;

  getMessagesThisMonth(): number;
  setMessagesThisMonth(value: number): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  hasLastMessageAt(): boolean;
  clearLastMessageAt(): void;
  getLastMessageAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastMessageAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getEndpointMessageCountsMap(): jspb.Map<string, number>;
  clearEndpointMessageCountsMap(): void;
  getStatus(): string;
  setStatus(value: string): void;

  hasLastSeenAt(): boolean;
  clearLastSeenAt(): void;
  getLastSeenAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastSeenAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationStatsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationStatsResponse): GetBaseStationStatsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationStatsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationStatsResponse;
  static deserializeBinaryFromReader(message: GetBaseStationStatsResponse, reader: jspb.BinaryReader): GetBaseStationStatsResponse;
}

export namespace GetBaseStationStatsResponse {
  export type AsObject = {
    bseui: string,
    totalMessages: number,
    totalEndpoints: number,
    messagesToday: number,
    messagesThisWeek: number,
    messagesThisMonth: number,
    avgRssi: number,
    avgSnr: number,
    lastMessageAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endpointMessageCountsMap: Array<[string, number]>,
    status: string,
    lastSeenAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetBaseStationAvailabilityRequest extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getIntervalSeconds(): number;
  setIntervalSeconds(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationAvailabilityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationAvailabilityRequest): GetBaseStationAvailabilityRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationAvailabilityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationAvailabilityRequest;
  static deserializeBinaryFromReader(message: GetBaseStationAvailabilityRequest, reader: jspb.BinaryReader): GetBaseStationAvailabilityRequest;
}

export namespace GetBaseStationAvailabilityRequest {
  export type AsObject = {
    bseui: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    intervalSeconds: number,
  }
}

export class GetBaseStationAvailabilityResponse extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  clearAvailabilityList(): void;
  getAvailabilityList(): Array<number>;
  setAvailabilityList(value: Array<number>): void;
  addAvailability(value: number, index?: number): number;

  getIntervalSeconds(): number;
  setIntervalSeconds(value: number): void;

  hasLastPointTimestamp(): boolean;
  clearLastPointTimestamp(): void;
  getLastPointTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastPointTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationAvailabilityResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationAvailabilityResponse): GetBaseStationAvailabilityResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationAvailabilityResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationAvailabilityResponse;
  static deserializeBinaryFromReader(message: GetBaseStationAvailabilityResponse, reader: jspb.BinaryReader): GetBaseStationAvailabilityResponse;
}

export namespace GetBaseStationAvailabilityResponse {
  export type AsObject = {
    bseui: string,
    availabilityList: Array<number>,
    intervalSeconds: number,
    lastPointTimestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetBaseStationMessagesReceivedRequest extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getIntervalSeconds(): number;
  setIntervalSeconds(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessagesReceivedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessagesReceivedRequest): GetBaseStationMessagesReceivedRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessagesReceivedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessagesReceivedRequest;
  static deserializeBinaryFromReader(message: GetBaseStationMessagesReceivedRequest, reader: jspb.BinaryReader): GetBaseStationMessagesReceivedRequest;
}

export namespace GetBaseStationMessagesReceivedRequest {
  export type AsObject = {
    bseui: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    intervalSeconds: number,
  }
}

export class GetBaseStationMessagesReceivedResponse extends jspb.Message {
  getBseui(): string;
  setBseui(value: string): void;

  clearReceivedList(): void;
  getReceivedList(): Array<number>;
  setReceivedList(value: Array<number>): void;
  addReceived(value: number, index?: number): number;

  getIntervalSeconds(): number;
  setIntervalSeconds(value: number): void;

  hasLastPointTimestamp(): boolean;
  clearLastPointTimestamp(): void;
  getLastPointTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastPointTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessagesReceivedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessagesReceivedResponse): GetBaseStationMessagesReceivedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessagesReceivedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessagesReceivedResponse;
  static deserializeBinaryFromReader(message: GetBaseStationMessagesReceivedResponse, reader: jspb.BinaryReader): GetBaseStationMessagesReceivedResponse;
}

export namespace GetBaseStationMessagesReceivedResponse {
  export type AsObject = {
    bseui: string,
    receivedList: Array<number>,
    intervalSeconds: number,
    lastPointTimestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetMessageRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetMessageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetMessageRequest): GetMessageRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetMessageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetMessageRequest;
  static deserializeBinaryFromReader(message: GetMessageRequest, reader: jspb.BinaryReader): GetMessageRequest;
}

export namespace GetMessageRequest {
  export type AsObject = {
    id: string,
    tenantId: string,
  }
}

export class SendDownlinkRequest extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  clearPayloadsList(): void;
  getPayloadsList(): Array<Uint8Array | string>;
  getPayloadsList_asU8(): Array<Uint8Array>;
  getPayloadsList_asB64(): Array<string>;
  setPayloadsList(value: Array<Uint8Array | string>): void;
  addPayloads(value: Uint8Array | string, index?: number): Uint8Array | string;

  getPriority(): number;
  setPriority(value: number): void;

  getCntDepend(): boolean;
  setCntDepend(value: boolean): void;

  clearPacketCntList(): void;
  getPacketCntList(): Array<number>;
  setPacketCntList(value: Array<number>): void;
  addPacketCnt(value: number, index?: number): number;

  getFormat(): number;
  setFormat(value: number): void;

  getResponseExp(): boolean;
  setResponseExp(value: boolean): void;

  getResponsePrio(): boolean;
  setResponsePrio(value: boolean): void;

  getDlWindReq(): boolean;
  setDlWindReq(value: boolean): void;

  getExpOnly(): boolean;
  setExpOnly(value: boolean): void;

  getDlRxStatQry(): boolean;
  setDlRxStatQry(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SendDownlinkRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SendDownlinkRequest): SendDownlinkRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SendDownlinkRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SendDownlinkRequest;
  static deserializeBinaryFromReader(message: SendDownlinkRequest, reader: jspb.BinaryReader): SendDownlinkRequest;
}

export namespace SendDownlinkRequest {
  export type AsObject = {
    epeui: string,
    tenantId: string,
    payloadsList: Array<Uint8Array | string>,
    priority: number,
    cntDepend: boolean,
    packetCntList: Array<number>,
    format: number,
    responseExp: boolean,
    responsePrio: boolean,
    dlWindReq: boolean,
    expOnly: boolean,
    dlRxStatQry: boolean,
  }
}

export class SendDownlinkResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SendDownlinkResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SendDownlinkResponse): SendDownlinkResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SendDownlinkResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SendDownlinkResponse;
  static deserializeBinaryFromReader(message: SendDownlinkResponse, reader: jspb.BinaryReader): SendDownlinkResponse;
}

export namespace SendDownlinkResponse {
  export type AsObject = {
    id: string,
    status: string,
  }
}

export class RevokeDownlinkRequest extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getQueueId(): string;
  setQueueId(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RevokeDownlinkRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RevokeDownlinkRequest): RevokeDownlinkRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RevokeDownlinkRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RevokeDownlinkRequest;
  static deserializeBinaryFromReader(message: RevokeDownlinkRequest, reader: jspb.BinaryReader): RevokeDownlinkRequest;
}

export namespace RevokeDownlinkRequest {
  export type AsObject = {
    epeui: string,
    queueId: string,
    tenantId: string,
  }
}

export class RevokeDownlinkResponse extends jspb.Message {
  getStatus(): string;
  setStatus(value: string): void;

  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RevokeDownlinkResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RevokeDownlinkResponse): RevokeDownlinkResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RevokeDownlinkResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RevokeDownlinkResponse;
  static deserializeBinaryFromReader(message: RevokeDownlinkResponse, reader: jspb.BinaryReader): RevokeDownlinkResponse;
}

export namespace RevokeDownlinkResponse {
  export type AsObject = {
    status: string,
    message: string,
  }
}

export class ListDownlinkQueueRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): void;

  getEpeui(): string;
  setEpeui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDownlinkQueueRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListDownlinkQueueRequest): ListDownlinkQueueRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDownlinkQueueRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDownlinkQueueRequest;
  static deserializeBinaryFromReader(message: ListDownlinkQueueRequest, reader: jspb.BinaryReader): ListDownlinkQueueRequest;
}

export namespace ListDownlinkQueueRequest {
  export type AsObject = {
    tenantId: string,
    epeui: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListDownlinkQueueResponse extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<DownlinkMessage>;
  setMessagesList(value: Array<DownlinkMessage>): void;
  addMessages(value?: DownlinkMessage, index?: number): DownlinkMessage;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDownlinkQueueResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListDownlinkQueueResponse): ListDownlinkQueueResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDownlinkQueueResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDownlinkQueueResponse;
  static deserializeBinaryFromReader(message: ListDownlinkQueueResponse, reader: jspb.BinaryReader): ListDownlinkQueueResponse;
}

export namespace ListDownlinkQueueResponse {
  export type AsObject = {
    messagesList: Array<DownlinkMessage.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class DownlinkMessage extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEpeui(): string;
  setEpeui(value: string): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): void;

  getPriority(): number;
  setPriority(value: number): void;

  getStatus(): string;
  setStatus(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasScheduledAt(): boolean;
  clearScheduledAt(): void;
  getScheduledAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setScheduledAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasTransmittedAt(): boolean;
  clearTransmittedAt(): void;
  getTransmittedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTransmittedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getResult(): string;
  setResult(value: string): void;

  getTxTime(): number;
  setTxTime(value: number): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  getCntDepend(): boolean;
  setCntDepend(value: boolean): void;

  clearPacketCntList(): void;
  getPacketCntList(): Array<number>;
  setPacketCntList(value: Array<number>): void;
  addPacketCnt(value: number, index?: number): number;

  getFormat(): number;
  setFormat(value: number): void;

  getResponseExp(): boolean;
  setResponseExp(value: boolean): void;

  getResponsePrio(): boolean;
  setResponsePrio(value: boolean): void;

  getDlWindReq(): boolean;
  setDlWindReq(value: boolean): void;

  getExpOnly(): boolean;
  setExpOnly(value: boolean): void;

  getQueId(): number;
  setQueId(value: number): void;

  getAttempts(): number;
  setAttempts(value: number): void;

  getMaxAttempts(): number;
  setMaxAttempts(value: number): void;

  getTransmissionPacketCnt(): number;
  setTransmissionPacketCnt(value: number): void;

  clearPayloadsList(): void;
  getPayloadsList(): Array<Uint8Array | string>;
  getPayloadsList_asU8(): Array<Uint8Array>;
  getPayloadsList_asB64(): Array<string>;
  setPayloadsList(value: Array<Uint8Array | string>): void;
  addPayloads(value: Uint8Array | string, index?: number): Uint8Array | string;

  getDlRxStatQry(): boolean;
  setDlRxStatQry(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownlinkMessage.AsObject;
  static toObject(includeInstance: boolean, msg: DownlinkMessage): DownlinkMessage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DownlinkMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownlinkMessage;
  static deserializeBinaryFromReader(message: DownlinkMessage, reader: jspb.BinaryReader): DownlinkMessage;
}

export namespace DownlinkMessage {
  export type AsObject = {
    id: string,
    epeui: string,
    tenantId: string,
    payload: Uint8Array | string,
    priority: number,
    status: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    scheduledAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    transmittedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    result: string,
    txTime: number,
    bsEui: string,
    cntDepend: boolean,
    packetCntList: Array<number>,
    format: number,
    responseExp: boolean,
    responsePrio: boolean,
    dlWindReq: boolean,
    expOnly: boolean,
    queId: number,
    attempts: number,
    maxAttempts: number,
    transmissionPacketCnt: number,
    payloadsList: Array<Uint8Array | string>,
    dlRxStatQry: boolean,
  }
}

export class GetDownlinkResultsRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): void;

  getEpeui(): string;
  setEpeui(value: string): void;

  getStatusFilter(): string;
  setStatusFilter(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasTimeFrom(): boolean;
  clearTimeFrom(): void;
  getTimeFrom(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimeFrom(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasTimeTo(): boolean;
  clearTimeTo(): void;
  getTimeTo(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimeTo(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownlinkResultsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownlinkResultsRequest): GetDownlinkResultsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDownlinkResultsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownlinkResultsRequest;
  static deserializeBinaryFromReader(message: GetDownlinkResultsRequest, reader: jspb.BinaryReader): GetDownlinkResultsRequest;
}

export namespace GetDownlinkResultsRequest {
  export type AsObject = {
    tenantId: string,
    epeui: string,
    statusFilter: string,
    pageSize: number,
    pageToken: string,
    timeFrom?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    timeTo?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetDownlinkResultsResponse extends jspb.Message {
  clearResultsList(): void;
  getResultsList(): Array<DownlinkMessage>;
  setResultsList(value: Array<DownlinkMessage>): void;
  addResults(value?: DownlinkMessage, index?: number): DownlinkMessage;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownlinkResultsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownlinkResultsResponse): GetDownlinkResultsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDownlinkResultsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownlinkResultsResponse;
  static deserializeBinaryFromReader(message: GetDownlinkResultsResponse, reader: jspb.BinaryReader): GetDownlinkResultsResponse;
}

export namespace GetDownlinkResultsResponse {
  export type AsObject = {
    resultsList: Array<DownlinkMessage.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class SendULTransmitRequest extends jspb.Message {
  getEpeui(): string;
  setEpeui(value: string): void;

  getUserData(): Uint8Array | string;
  getUserData_asU8(): Uint8Array;
  getUserData_asB64(): string;
  setUserData(value: Uint8Array | string): void;

  getPacketCnt(): number;
  setPacketCnt(value: number): void;

  getNwkSnKey(): Uint8Array | string;
  getNwkSnKey_asU8(): Uint8Array;
  getNwkSnKey_asB64(): string;
  setNwkSnKey(value: Uint8Array | string): void;

  getShAddr(): number;
  setShAddr(value: number): void;

  getProfile(): string;
  setProfile(value: string): void;

  getFormat(): number;
  setFormat(value: number): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SendULTransmitRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SendULTransmitRequest): SendULTransmitRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SendULTransmitRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SendULTransmitRequest;
  static deserializeBinaryFromReader(message: SendULTransmitRequest, reader: jspb.BinaryReader): SendULTransmitRequest;
}

export namespace SendULTransmitRequest {
  export type AsObject = {
    epeui: string,
    userData: Uint8Array | string,
    packetCnt: number,
    nwkSnKey: Uint8Array | string,
    shAddr: number,
    profile: string,
    format: number,
    tenantId: string,
    bsEui: string,
  }
}

export class SendULTransmitResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SendULTransmitResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SendULTransmitResponse): SendULTransmitResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SendULTransmitResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SendULTransmitResponse;
  static deserializeBinaryFromReader(message: SendULTransmitResponse, reader: jspb.BinaryReader): SendULTransmitResponse;
}

export namespace SendULTransmitResponse {
  export type AsObject = {
    id: string,
    status: string,
    message: string,
  }
}

export class BaseStationStatusRequest extends jspb.Message {
  getBsEui(): number;
  setBsEui(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationStatusRequest): BaseStationStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationStatusRequest;
  static deserializeBinaryFromReader(message: BaseStationStatusRequest, reader: jspb.BinaryReader): BaseStationStatusRequest;
}

export namespace BaseStationStatusRequest {
  export type AsObject = {
    bsEui: number,
  }
}

export class BaseStationStatusResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getMessage(): string;
  setMessage(value: string): void;

  getOpId(): number;
  setOpId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationStatusResponse): BaseStationStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationStatusResponse;
  static deserializeBinaryFromReader(message: BaseStationStatusResponse, reader: jspb.BinaryReader): BaseStationStatusResponse;
}

export namespace BaseStationStatusResponse {
  export type AsObject = {
    success: boolean,
    message: string,
    opId: number,
  }
}

export class InitiatePingRequest extends jspb.Message {
  getBsEui(): number;
  setBsEui(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InitiatePingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: InitiatePingRequest): InitiatePingRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: InitiatePingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InitiatePingRequest;
  static deserializeBinaryFromReader(message: InitiatePingRequest, reader: jspb.BinaryReader): InitiatePingRequest;
}

export namespace InitiatePingRequest {
  export type AsObject = {
    bsEui: number,
  }
}

export class InitiatePingResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getMessage(): string;
  setMessage(value: string): void;

  getOpId(): number;
  setOpId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InitiatePingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: InitiatePingResponse): InitiatePingResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: InitiatePingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InitiatePingResponse;
  static deserializeBinaryFromReader(message: InitiatePingResponse, reader: jspb.BinaryReader): InitiatePingResponse;
}

export namespace InitiatePingResponse {
  export type AsObject = {
    success: boolean,
    message: string,
    opId: number,
  }
}

export class ServiceStatus extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getUrl(): string;
  setUrl(value: string): void;

  getHealthy(): boolean;
  setHealthy(value: boolean): void;

  getLatencyMs(): number;
  setLatencyMs(value: number): void;

  getError(): string;
  setError(value: string): void;

  hasCheckedAt(): boolean;
  clearCheckedAt(): void;
  getCheckedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCheckedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServiceStatus.AsObject;
  static toObject(includeInstance: boolean, msg: ServiceStatus): ServiceStatus.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServiceStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServiceStatus;
  static deserializeBinaryFromReader(message: ServiceStatus, reader: jspb.BinaryReader): ServiceStatus;
}

export namespace ServiceStatus {
  export type AsObject = {
    name: string,
    url: string,
    healthy: boolean,
    latencyMs: number,
    error: string,
    checkedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class SystemStatus extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  hasUptime(): boolean;
  clearUptime(): void;
  getUptime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUptime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getActiveEndpoints(): number;
  setActiveEndpoints(value: number): void;

  getActiveBasestations(): number;
  setActiveBasestations(value: number): void;

  getMessagesProcessed(): number;
  setMessagesProcessed(value: number): void;

  clearServicesList(): void;
  getServicesList(): Array<ServiceStatus>;
  setServicesList(value: Array<ServiceStatus>): void;
  addServices(value?: ServiceStatus, index?: number): ServiceStatus;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SystemStatus.AsObject;
  static toObject(includeInstance: boolean, msg: SystemStatus): SystemStatus.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SystemStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SystemStatus;
  static deserializeBinaryFromReader(message: SystemStatus, reader: jspb.BinaryReader): SystemStatus;
}

export namespace SystemStatus {
  export type AsObject = {
    version: string,
    status: string,
    uptime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    activeEndpoints: number,
    activeBasestations: number,
    messagesProcessed: number,
    servicesList: Array<ServiceStatus.AsObject>,
  }
}

export class ReleaseInfo extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): void;

  getBuildTime(): string;
  setBuildTime(value: string): void;

  getGitCommit(): string;
  setGitCommit(value: string): void;

  getGitBranch(): string;
  setGitBranch(value: string): void;

  getBuildUser(): string;
  setBuildUser(value: string): void;

  getGoVersion(): string;
  setGoVersion(value: string): void;

  getSchemaVersion(): number;
  setSchemaVersion(value: number): void;

  getArtifactsMap(): jspb.Map<string, string>;
  clearArtifactsMap(): void;
  getScEui(): number;
  setScEui(value: number): void;

  getScVendor(): string;
  setScVendor(value: string): void;

  getScModel(): string;
  setScModel(value: string): void;

  getScName(): string;
  setScName(value: string): void;

  getScSwVersion(): string;
  setScSwVersion(value: string): void;

  getEdition(): string;
  setEdition(value: string): void;

  getLicenseId(): string;
  setLicenseId(value: string): void;

  getLicenseUrl(): string;
  setLicenseUrl(value: string): void;

  getSourceUrl(): string;
  setSourceUrl(value: string): void;

  getDocumentationUrl(): string;
  setDocumentationUrl(value: string): void;

  getHomepageUrl(): string;
  setHomepageUrl(value: string): void;

  getTrademarkNotice(): string;
  setTrademarkNotice(value: string): void;

  getEditionCode(): string;
  setEditionCode(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReleaseInfo.AsObject;
  static toObject(includeInstance: boolean, msg: ReleaseInfo): ReleaseInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReleaseInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReleaseInfo;
  static deserializeBinaryFromReader(message: ReleaseInfo, reader: jspb.BinaryReader): ReleaseInfo;
}

export namespace ReleaseInfo {
  export type AsObject = {
    version: string,
    buildTime: string,
    gitCommit: string,
    gitBranch: string,
    buildUser: string,
    goVersion: string,
    schemaVersion: number,
    artifactsMap: Array<[string, string]>,
    scEui: number,
    scVendor: string,
    scModel: string,
    scName: string,
    scSwVersion: string,
    edition: string,
    licenseId: string,
    licenseUrl: string,
    sourceUrl: string,
    documentationUrl: string,
    homepageUrl: string,
    trademarkNotice: string,
    editionCode: string,
  }
}

export class GetStatisticsRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getGranularity(): string;
  setGranularity(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetStatisticsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetStatisticsRequest): GetStatisticsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetStatisticsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetStatisticsRequest;
  static deserializeBinaryFromReader(message: GetStatisticsRequest, reader: jspb.BinaryReader): GetStatisticsRequest;
}

export namespace GetStatisticsRequest {
  export type AsObject = {
    tenantId: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    granularity: string,
  }
}

export class Statistics extends jspb.Message {
  getTotalMessages(): number;
  setTotalMessages(value: number): void;

  getTotalEndpoints(): number;
  setTotalEndpoints(value: number): void;

  getTotalBasestations(): number;
  setTotalBasestations(value: number): void;

  clearMessageCountsList(): void;
  getMessageCountsList(): Array<TimeSeriesData>;
  setMessageCountsList(value: Array<TimeSeriesData>): void;
  addMessageCounts(value?: TimeSeriesData, index?: number): TimeSeriesData;

  getEndpointMessageCountsMap(): jspb.Map<string, number>;
  clearEndpointMessageCountsMap(): void;
  getBasestationMessageCountsMap(): jspb.Map<string, number>;
  clearBasestationMessageCountsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Statistics.AsObject;
  static toObject(includeInstance: boolean, msg: Statistics): Statistics.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Statistics, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Statistics;
  static deserializeBinaryFromReader(message: Statistics, reader: jspb.BinaryReader): Statistics;
}

export namespace Statistics {
  export type AsObject = {
    totalMessages: number,
    totalEndpoints: number,
    totalBasestations: number,
    messageCountsList: Array<TimeSeriesData.AsObject>,
    endpointMessageCountsMap: Array<[string, number]>,
    basestationMessageCountsMap: Array<[string, number]>,
  }
}

export class TimeSeriesData extends jspb.Message {
  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getValue(): number;
  setValue(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TimeSeriesData.AsObject;
  static toObject(includeInstance: boolean, msg: TimeSeriesData): TimeSeriesData.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TimeSeriesData, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TimeSeriesData;
  static deserializeBinaryFromReader(message: TimeSeriesData, reader: jspb.BinaryReader): TimeSeriesData;
}

export namespace TimeSeriesData {
  export type AsObject = {
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    value: number,
  }
}

export class DLRXStatus extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  getRxTime(): number;
  setRxTime(value: number): void;

  getPacketCnt(): number;
  setPacketCnt(value: number): void;

  getDlRxSnr(): number;
  setDlRxSnr(value: number): void;

  getDlRxRssi(): number;
  setDlRxRssi(value: number): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DLRXStatus.AsObject;
  static toObject(includeInstance: boolean, msg: DLRXStatus): DLRXStatus.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DLRXStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DLRXStatus;
  static deserializeBinaryFromReader(message: DLRXStatus, reader: jspb.BinaryReader): DLRXStatus;
}

export namespace DLRXStatus {
  export type AsObject = {
    epEui: string,
    bsEui: string,
    rxTime: number,
    packetCnt: number,
    dlRxSnr: number,
    dlRxRssi: number,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetDLRXStatusRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getLimit(): number;
  setLimit(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDLRXStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDLRXStatusRequest): GetDLRXStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDLRXStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDLRXStatusRequest;
  static deserializeBinaryFromReader(message: GetDLRXStatusRequest, reader: jspb.BinaryReader): GetDLRXStatusRequest;
}

export namespace GetDLRXStatusRequest {
  export type AsObject = {
    epEui: string,
    limit: number,
    offset: number,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetDLRXStatusResponse extends jspb.Message {
  clearStatusesList(): void;
  getStatusesList(): Array<DLRXStatus>;
  setStatusesList(value: Array<DLRXStatus>): void;
  addStatuses(value?: DLRXStatus, index?: number): DLRXStatus;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDLRXStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetDLRXStatusResponse): GetDLRXStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDLRXStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDLRXStatusResponse;
  static deserializeBinaryFromReader(message: GetDLRXStatusResponse, reader: jspb.BinaryReader): GetDLRXStatusResponse;
}

export namespace GetDLRXStatusResponse {
  export type AsObject = {
    statusesList: Array<DLRXStatus.AsObject>,
    totalCount: number,
    avgSnr: number,
    avgRssi: number,
  }
}

export class QueryDLRXStatusRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryDLRXStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: QueryDLRXStatusRequest): QueryDLRXStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryDLRXStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryDLRXStatusRequest;
  static deserializeBinaryFromReader(message: QueryDLRXStatusRequest, reader: jspb.BinaryReader): QueryDLRXStatusRequest;
}

export namespace QueryDLRXStatusRequest {
  export type AsObject = {
    epEui: string,
  }
}

export class QueryDLRXStatusResponse extends jspb.Message {
  getQueryInitiated(): boolean;
  setQueryInitiated(value: boolean): void;

  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): QueryDLRXStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: QueryDLRXStatusResponse): QueryDLRXStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: QueryDLRXStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): QueryDLRXStatusResponse;
  static deserializeBinaryFromReader(message: QueryDLRXStatusResponse, reader: jspb.BinaryReader): QueryDLRXStatusResponse;
}

export namespace QueryDLRXStatusResponse {
  export type AsObject = {
    queryInitiated: boolean,
    message: string,
  }
}

export class DLRXStatusQuery extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  getOpId(): number;
  setOpId(value: number): void;

  getStatus(): string;
  setStatus(value: string): void;

  hasRequestedAt(): boolean;
  clearRequestedAt(): void;
  getRequestedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRequestedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasReceivedAt(): boolean;
  clearReceivedAt(): void;
  getReceivedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setReceivedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getOrgUuid(): string;
  setOrgUuid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DLRXStatusQuery.AsObject;
  static toObject(includeInstance: boolean, msg: DLRXStatusQuery): DLRXStatusQuery.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DLRXStatusQuery, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DLRXStatusQuery;
  static deserializeBinaryFromReader(message: DLRXStatusQuery, reader: jspb.BinaryReader): DLRXStatusQuery;
}

export namespace DLRXStatusQuery {
  export type AsObject = {
    epEui: string,
    bsEui: string,
    opId: number,
    status: string,
    requestedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    receivedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    orgUuid: string,
  }
}

export class GetDLRXStatusQueriesRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getLimit(): number;
  setLimit(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDLRXStatusQueriesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDLRXStatusQueriesRequest): GetDLRXStatusQueriesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDLRXStatusQueriesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDLRXStatusQueriesRequest;
  static deserializeBinaryFromReader(message: GetDLRXStatusQueriesRequest, reader: jspb.BinaryReader): GetDLRXStatusQueriesRequest;
}

export namespace GetDLRXStatusQueriesRequest {
  export type AsObject = {
    epEui: string,
    limit: number,
    offset: number,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class DLRXStatusQueryStats extends jspb.Message {
  getPending(): number;
  setPending(value: number): void;

  getReceived(): number;
  setReceived(value: number): void;

  getTimeout(): number;
  setTimeout(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DLRXStatusQueryStats.AsObject;
  static toObject(includeInstance: boolean, msg: DLRXStatusQueryStats): DLRXStatusQueryStats.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DLRXStatusQueryStats, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DLRXStatusQueryStats;
  static deserializeBinaryFromReader(message: DLRXStatusQueryStats, reader: jspb.BinaryReader): DLRXStatusQueryStats;
}

export namespace DLRXStatusQueryStats {
  export type AsObject = {
    pending: number,
    received: number,
    timeout: number,
  }
}

export class GetDLRXStatusQueriesResponse extends jspb.Message {
  clearQueriesList(): void;
  getQueriesList(): Array<DLRXStatusQuery>;
  setQueriesList(value: Array<DLRXStatusQuery>): void;
  addQueries(value?: DLRXStatusQuery, index?: number): DLRXStatusQuery;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  hasStats(): boolean;
  clearStats(): void;
  getStats(): DLRXStatusQueryStats | undefined;
  setStats(value?: DLRXStatusQueryStats): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDLRXStatusQueriesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetDLRXStatusQueriesResponse): GetDLRXStatusQueriesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDLRXStatusQueriesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDLRXStatusQueriesResponse;
  static deserializeBinaryFromReader(message: GetDLRXStatusQueriesResponse, reader: jspb.BinaryReader): GetDLRXStatusQueriesResponse;
}

export namespace GetDLRXStatusQueriesResponse {
  export type AsObject = {
    queriesList: Array<DLRXStatusQuery.AsObject>,
    totalCount: number,
    stats?: DLRXStatusQueryStats.AsObject,
  }
}

export class Integration extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getType(): string;
  setType(value: string): void;

  hasConfig(): boolean;
  clearConfig(): void;
  getConfig(): google_protobuf_struct_pb.Struct | undefined;
  setConfig(value?: google_protobuf_struct_pb.Struct): void;

  hasEventFilter(): boolean;
  clearEventFilter(): void;
  getEventFilter(): google_protobuf_struct_pb.Struct | undefined;
  setEventFilter(value?: google_protobuf_struct_pb.Struct): void;

  getDeliveryFormat(): string;
  setDeliveryFormat(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Integration.AsObject;
  static toObject(includeInstance: boolean, msg: Integration): Integration.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Integration, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Integration;
  static deserializeBinaryFromReader(message: Integration, reader: jspb.BinaryReader): Integration;
}

export namespace Integration {
  export type AsObject = {
    id: number,
    name: string,
    description: string,
    type: string,
    config?: google_protobuf_struct_pb.Struct.AsObject,
    eventFilter?: google_protobuf_struct_pb.Struct.AsObject,
    deliveryFormat: string,
    status: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class CreateIntegrationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getType(): string;
  setType(value: string): void;

  hasConfig(): boolean;
  clearConfig(): void;
  getConfig(): google_protobuf_struct_pb.Struct | undefined;
  setConfig(value?: google_protobuf_struct_pb.Struct): void;

  hasEventFilter(): boolean;
  clearEventFilter(): void;
  getEventFilter(): google_protobuf_struct_pb.Struct | undefined;
  setEventFilter(value?: google_protobuf_struct_pb.Struct): void;

  getDeliveryFormat(): string;
  setDeliveryFormat(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateIntegrationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateIntegrationRequest): CreateIntegrationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateIntegrationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateIntegrationRequest;
  static deserializeBinaryFromReader(message: CreateIntegrationRequest, reader: jspb.BinaryReader): CreateIntegrationRequest;
}

export namespace CreateIntegrationRequest {
  export type AsObject = {
    name: string,
    description: string,
    type: string,
    config?: google_protobuf_struct_pb.Struct.AsObject,
    eventFilter?: google_protobuf_struct_pb.Struct.AsObject,
    deliveryFormat: string,
  }
}

export class GetIntegrationRequest extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetIntegrationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetIntegrationRequest): GetIntegrationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetIntegrationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetIntegrationRequest;
  static deserializeBinaryFromReader(message: GetIntegrationRequest, reader: jspb.BinaryReader): GetIntegrationRequest;
}

export namespace GetIntegrationRequest {
  export type AsObject = {
    id: number,
  }
}

export class UpdateIntegrationRequest extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  hasConfig(): boolean;
  clearConfig(): void;
  getConfig(): google_protobuf_struct_pb.Struct | undefined;
  setConfig(value?: google_protobuf_struct_pb.Struct): void;

  hasEventFilter(): boolean;
  clearEventFilter(): void;
  getEventFilter(): google_protobuf_struct_pb.Struct | undefined;
  setEventFilter(value?: google_protobuf_struct_pb.Struct): void;

  getStatus(): string;
  setStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateIntegrationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateIntegrationRequest): UpdateIntegrationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateIntegrationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateIntegrationRequest;
  static deserializeBinaryFromReader(message: UpdateIntegrationRequest, reader: jspb.BinaryReader): UpdateIntegrationRequest;
}

export namespace UpdateIntegrationRequest {
  export type AsObject = {
    id: number,
    name: string,
    description: string,
    config?: google_protobuf_struct_pb.Struct.AsObject,
    eventFilter?: google_protobuf_struct_pb.Struct.AsObject,
    status: string,
  }
}

export class DeleteIntegrationRequest extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteIntegrationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteIntegrationRequest): DeleteIntegrationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteIntegrationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteIntegrationRequest;
  static deserializeBinaryFromReader(message: DeleteIntegrationRequest, reader: jspb.BinaryReader): DeleteIntegrationRequest;
}

export namespace DeleteIntegrationRequest {
  export type AsObject = {
    id: number,
  }
}

export class ListIntegrationsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListIntegrationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListIntegrationsRequest): ListIntegrationsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListIntegrationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListIntegrationsRequest;
  static deserializeBinaryFromReader(message: ListIntegrationsRequest, reader: jspb.BinaryReader): ListIntegrationsRequest;
}

export namespace ListIntegrationsRequest {
  export type AsObject = {
    pageSize: number,
    offset: number,
  }
}

export class ListIntegrationsResponse extends jspb.Message {
  clearIntegrationsList(): void;
  getIntegrationsList(): Array<Integration>;
  setIntegrationsList(value: Array<Integration>): void;
  addIntegrations(value?: Integration, index?: number): Integration;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListIntegrationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListIntegrationsResponse): ListIntegrationsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListIntegrationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListIntegrationsResponse;
  static deserializeBinaryFromReader(message: ListIntegrationsResponse, reader: jspb.BinaryReader): ListIntegrationsResponse;
}

export namespace ListIntegrationsResponse {
  export type AsObject = {
    integrationsList: Array<Integration.AsObject>,
    totalCount: number,
  }
}

export class GetAnalyticsOverviewRequest extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAnalyticsOverviewRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAnalyticsOverviewRequest): GetAnalyticsOverviewRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetAnalyticsOverviewRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAnalyticsOverviewRequest;
  static deserializeBinaryFromReader(message: GetAnalyticsOverviewRequest, reader: jspb.BinaryReader): GetAnalyticsOverviewRequest;
}

export namespace GetAnalyticsOverviewRequest {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetAnalyticsOverviewResponse extends jspb.Message {
  hasOverview(): boolean;
  clearOverview(): void;
  getOverview(): AnalyticsOverview | undefined;
  setOverview(value?: AnalyticsOverview): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAnalyticsOverviewResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetAnalyticsOverviewResponse): GetAnalyticsOverviewResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetAnalyticsOverviewResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAnalyticsOverviewResponse;
  static deserializeBinaryFromReader(message: GetAnalyticsOverviewResponse, reader: jspb.BinaryReader): GetAnalyticsOverviewResponse;
}

export namespace GetAnalyticsOverviewResponse {
  export type AsObject = {
    overview?: AnalyticsOverview.AsObject,
  }
}

export class GetActivityAnalyticsRequest extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getGranularity(): string;
  setGranularity(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetActivityAnalyticsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetActivityAnalyticsRequest): GetActivityAnalyticsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetActivityAnalyticsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetActivityAnalyticsRequest;
  static deserializeBinaryFromReader(message: GetActivityAnalyticsRequest, reader: jspb.BinaryReader): GetActivityAnalyticsRequest;
}

export namespace GetActivityAnalyticsRequest {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    granularity: string,
  }
}

export class GetActivityAnalyticsResponse extends jspb.Message {
  hasActivity(): boolean;
  clearActivity(): void;
  getActivity(): ActivityAnalytics | undefined;
  setActivity(value?: ActivityAnalytics): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetActivityAnalyticsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetActivityAnalyticsResponse): GetActivityAnalyticsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetActivityAnalyticsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetActivityAnalyticsResponse;
  static deserializeBinaryFromReader(message: GetActivityAnalyticsResponse, reader: jspb.BinaryReader): GetActivityAnalyticsResponse;
}

export namespace GetActivityAnalyticsResponse {
  export type AsObject = {
    activity?: ActivityAnalytics.AsObject,
  }
}

export class GetSignalQualityAnalyticsRequest extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSignalQualityAnalyticsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSignalQualityAnalyticsRequest): GetSignalQualityAnalyticsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetSignalQualityAnalyticsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSignalQualityAnalyticsRequest;
  static deserializeBinaryFromReader(message: GetSignalQualityAnalyticsRequest, reader: jspb.BinaryReader): GetSignalQualityAnalyticsRequest;
}

export namespace GetSignalQualityAnalyticsRequest {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetSignalQualityAnalyticsResponse extends jspb.Message {
  hasSignalQuality(): boolean;
  clearSignalQuality(): void;
  getSignalQuality(): SignalQualityAnalytics | undefined;
  setSignalQuality(value?: SignalQualityAnalytics): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSignalQualityAnalyticsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetSignalQualityAnalyticsResponse): GetSignalQualityAnalyticsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetSignalQualityAnalyticsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSignalQualityAnalyticsResponse;
  static deserializeBinaryFromReader(message: GetSignalQualityAnalyticsResponse, reader: jspb.BinaryReader): GetSignalQualityAnalyticsResponse;
}

export namespace GetSignalQualityAnalyticsResponse {
  export type AsObject = {
    signalQuality?: SignalQualityAnalytics.AsObject,
  }
}

export class AnalyticsOverview extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getTotalMessages(): number;
  setTotalMessages(value: number): void;

  getActiveEndpoints(): number;
  setActiveEndpoints(value: number): void;

  getActiveBaseStations(): number;
  setActiveBaseStations(value: number): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  hasFirstMessage(): boolean;
  clearFirstMessage(): void;
  getFirstMessage(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstMessage(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastMessage(): boolean;
  clearLastMessage(): void;
  getLastMessage(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastMessage(value?: google_protobuf_timestamp_pb.Timestamp): void;

  clearHourlyActivityList(): void;
  getHourlyActivityList(): Array<HourlyActivity>;
  setHourlyActivityList(value: Array<HourlyActivity>): void;
  addHourlyActivity(value?: HourlyActivity, index?: number): HourlyActivity;

  getMessagesLast24h(): number;
  setMessagesLast24h(value: number): void;

  getMessagesLastWeek(): number;
  setMessagesLastWeek(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AnalyticsOverview.AsObject;
  static toObject(includeInstance: boolean, msg: AnalyticsOverview): AnalyticsOverview.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AnalyticsOverview, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AnalyticsOverview;
  static deserializeBinaryFromReader(message: AnalyticsOverview, reader: jspb.BinaryReader): AnalyticsOverview;
}

export namespace AnalyticsOverview {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    totalMessages: number,
    activeEndpoints: number,
    activeBaseStations: number,
    avgRssi: number,
    avgSnr: number,
    firstMessage?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastMessage?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    hourlyActivityList: Array<HourlyActivity.AsObject>,
    messagesLast24h: number,
    messagesLastWeek: number,
  }
}

export class HourlyActivity extends jspb.Message {
  hasHour(): boolean;
  clearHour(): void;
  getHour(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setHour(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getMessageCount(): number;
  setMessageCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HourlyActivity.AsObject;
  static toObject(includeInstance: boolean, msg: HourlyActivity): HourlyActivity.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: HourlyActivity, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HourlyActivity;
  static deserializeBinaryFromReader(message: HourlyActivity, reader: jspb.BinaryReader): HourlyActivity;
}

export namespace HourlyActivity {
  export type AsObject = {
    hour?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    messageCount: number,
  }
}

export class ActivityAnalytics extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getTotalMessages(): number;
  setTotalMessages(value: number): void;

  getUniqueEndpoints(): number;
  setUniqueEndpoints(value: number): void;

  getUniqueBaseStations(): number;
  setUniqueBaseStations(value: number): void;

  clearTimeSlotsList(): void;
  getTimeSlotsList(): Array<TimeSlotActivity>;
  setTimeSlotsList(value: Array<TimeSlotActivity>): void;
  addTimeSlots(value?: TimeSlotActivity, index?: number): TimeSlotActivity;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ActivityAnalytics.AsObject;
  static toObject(includeInstance: boolean, msg: ActivityAnalytics): ActivityAnalytics.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ActivityAnalytics, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ActivityAnalytics;
  static deserializeBinaryFromReader(message: ActivityAnalytics, reader: jspb.BinaryReader): ActivityAnalytics;
}

export namespace ActivityAnalytics {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    totalMessages: number,
    uniqueEndpoints: number,
    uniqueBaseStations: number,
    timeSlotsList: Array<TimeSlotActivity.AsObject>,
  }
}

export class TimeSlotActivity extends jspb.Message {
  hasSlot(): boolean;
  clearSlot(): void;
  getSlot(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setSlot(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getMessageCount(): number;
  setMessageCount(value: number): void;

  getEndpointCount(): number;
  setEndpointCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TimeSlotActivity.AsObject;
  static toObject(includeInstance: boolean, msg: TimeSlotActivity): TimeSlotActivity.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TimeSlotActivity, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TimeSlotActivity;
  static deserializeBinaryFromReader(message: TimeSlotActivity, reader: jspb.BinaryReader): TimeSlotActivity;
}

export namespace TimeSlotActivity {
  export type AsObject = {
    slot?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    messageCount: number,
    endpointCount: number,
  }
}

export class SignalQualityAnalytics extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasOverall(): boolean;
  clearOverall(): void;
  getOverall(): SignalQualityOverall | undefined;
  setOverall(value?: SignalQualityOverall): void;

  clearByBaseStationList(): void;
  getByBaseStationList(): Array<BaseStationSignalQuality>;
  setByBaseStationList(value: Array<BaseStationSignalQuality>): void;
  addByBaseStation(value?: BaseStationSignalQuality, index?: number): BaseStationSignalQuality;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SignalQualityAnalytics.AsObject;
  static toObject(includeInstance: boolean, msg: SignalQualityAnalytics): SignalQualityAnalytics.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SignalQualityAnalytics, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SignalQualityAnalytics;
  static deserializeBinaryFromReader(message: SignalQualityAnalytics, reader: jspb.BinaryReader): SignalQualityAnalytics;
}

export namespace SignalQualityAnalytics {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    overall?: SignalQualityOverall.AsObject,
    byBaseStationList: Array<BaseStationSignalQuality.AsObject>,
  }
}

export class SignalQualityOverall extends jspb.Message {
  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getMinRssi(): number;
  setMinRssi(value: number): void;

  getMaxRssi(): number;
  setMaxRssi(value: number): void;

  getMedianRssi(): number;
  setMedianRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  getMinSnr(): number;
  setMinSnr(value: number): void;

  getMaxSnr(): number;
  setMaxSnr(value: number): void;

  getMedianSnr(): number;
  setMedianSnr(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SignalQualityOverall.AsObject;
  static toObject(includeInstance: boolean, msg: SignalQualityOverall): SignalQualityOverall.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SignalQualityOverall, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SignalQualityOverall;
  static deserializeBinaryFromReader(message: SignalQualityOverall, reader: jspb.BinaryReader): SignalQualityOverall;
}

export namespace SignalQualityOverall {
  export type AsObject = {
    avgRssi: number,
    minRssi: number,
    maxRssi: number,
    medianRssi: number,
    avgSnr: number,
    minSnr: number,
    maxSnr: number,
    medianSnr: number,
  }
}

export class BaseStationSignalQuality extends jspb.Message {
  getEui(): string;
  setEui(value: string): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  getMessageCount(): number;
  setMessageCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationSignalQuality.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationSignalQuality): BaseStationSignalQuality.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationSignalQuality, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationSignalQuality;
  static deserializeBinaryFromReader(message: BaseStationSignalQuality, reader: jspb.BinaryReader): BaseStationSignalQuality;
}

export namespace BaseStationSignalQuality {
  export type AsObject = {
    eui: string,
    avgRssi: number,
    avgSnr: number,
    messageCount: number,
  }
}

export class ListEventsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  clearCategoriesList(): void;
  getCategoriesList(): Array<string>;
  setCategoriesList(value: Array<string>): void;
  addCategories(value: string, index?: number): string;

  getSeverity(): string;
  setSeverity(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  clearEventTypesList(): void;
  getEventTypesList(): Array<string>;
  setEventTypesList(value: Array<string>): void;
  addEventTypes(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListEventsRequest): ListEventsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEventsRequest;
  static deserializeBinaryFromReader(message: ListEventsRequest, reader: jspb.BinaryReader): ListEventsRequest;
}

export namespace ListEventsRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
    categoriesList: Array<string>,
    severity: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    eventTypesList: Array<string>,
  }
}

export class ListEventsResponse extends jspb.Message {
  clearEventsList(): void;
  getEventsList(): Array<Event>;
  setEventsList(value: Array<Event>): void;
  addEvents(value?: Event, index?: number): Event;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListEventsResponse): ListEventsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEventsResponse;
  static deserializeBinaryFromReader(message: ListEventsResponse, reader: jspb.BinaryReader): ListEventsResponse;
}

export namespace ListEventsResponse {
  export type AsObject = {
    eventsList: Array<Event.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class ListBaseStationActivityRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationActivityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationActivityRequest): ListBaseStationActivityRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationActivityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationActivityRequest;
  static deserializeBinaryFromReader(message: ListBaseStationActivityRequest, reader: jspb.BinaryReader): ListBaseStationActivityRequest;
}

export namespace ListBaseStationActivityRequest {
  export type AsObject = {
    bsEui: string,
    pageSize: number,
    pageToken: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListBaseStationActivityResponse extends jspb.Message {
  clearItemsList(): void;
  getItemsList(): Array<BaseStationActivityItem>;
  setItemsList(value: Array<BaseStationActivityItem>): void;
  addItems(value?: BaseStationActivityItem, index?: number): BaseStationActivityItem;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationActivityResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationActivityResponse): ListBaseStationActivityResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationActivityResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationActivityResponse;
  static deserializeBinaryFromReader(message: ListBaseStationActivityResponse, reader: jspb.BinaryReader): ListBaseStationActivityResponse;
}

export namespace ListBaseStationActivityResponse {
  export type AsObject = {
    itemsList: Array<BaseStationActivityItem.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class BaseStationActivityItem extends jspb.Message {
  hasOccurredAt(): boolean;
  clearOccurredAt(): void;
  getOccurredAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setOccurredAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEvent(): boolean;
  clearEvent(): void;
  getEvent(): Event | undefined;
  setEvent(value?: Event): void;

  hasMessage(): boolean;
  clearMessage(): void;
  getMessage(): BaseStationMessage | undefined;
  setMessage(value?: BaseStationMessage): void;

  getItemCase(): BaseStationActivityItem.ItemCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationActivityItem.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationActivityItem): BaseStationActivityItem.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationActivityItem, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationActivityItem;
  static deserializeBinaryFromReader(message: BaseStationActivityItem, reader: jspb.BinaryReader): BaseStationActivityItem;
}

export namespace BaseStationActivityItem {
  export type AsObject = {
    occurredAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    event?: Event.AsObject,
    message?: BaseStationMessage.AsObject,
  }

  export enum ItemCase {
    ITEM_NOT_SET = 0,
    EVENT = 2,
    MESSAGE = 3,
  }
}

export class ListEndpointActivityRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndpointActivityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndpointActivityRequest): ListEndpointActivityRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndpointActivityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndpointActivityRequest;
  static deserializeBinaryFromReader(message: ListEndpointActivityRequest, reader: jspb.BinaryReader): ListEndpointActivityRequest;
}

export namespace ListEndpointActivityRequest {
  export type AsObject = {
    epEui: string,
    pageSize: number,
    pageToken: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListEndpointActivityResponse extends jspb.Message {
  clearItemsList(): void;
  getItemsList(): Array<EndpointActivityItem>;
  setItemsList(value: Array<EndpointActivityItem>): void;
  addItems(value?: EndpointActivityItem, index?: number): EndpointActivityItem;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndpointActivityResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndpointActivityResponse): ListEndpointActivityResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndpointActivityResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndpointActivityResponse;
  static deserializeBinaryFromReader(message: ListEndpointActivityResponse, reader: jspb.BinaryReader): ListEndpointActivityResponse;
}

export namespace ListEndpointActivityResponse {
  export type AsObject = {
    itemsList: Array<EndpointActivityItem.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class EndpointActivityItem extends jspb.Message {
  hasOccurredAt(): boolean;
  clearOccurredAt(): void;
  getOccurredAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setOccurredAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEvent(): boolean;
  clearEvent(): void;
  getEvent(): Event | undefined;
  setEvent(value?: Event): void;

  hasMessage(): boolean;
  clearMessage(): void;
  getMessage(): Message | undefined;
  setMessage(value?: Message): void;

  getItemCase(): EndpointActivityItem.ItemCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EndpointActivityItem.AsObject;
  static toObject(includeInstance: boolean, msg: EndpointActivityItem): EndpointActivityItem.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EndpointActivityItem, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EndpointActivityItem;
  static deserializeBinaryFromReader(message: EndpointActivityItem, reader: jspb.BinaryReader): EndpointActivityItem;
}

export namespace EndpointActivityItem {
  export type AsObject = {
    occurredAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    event?: Event.AsObject,
    message?: Message.AsObject,
  }

  export enum ItemCase {
    ITEM_NOT_SET = 0,
    EVENT = 2,
    MESSAGE = 3,
  }
}

export class StreamEventsRequest extends jspb.Message {
  getCategory(): string;
  setCategory(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreamEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StreamEventsRequest): StreamEventsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StreamEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreamEventsRequest;
  static deserializeBinaryFromReader(message: StreamEventsRequest, reader: jspb.BinaryReader): StreamEventsRequest;
}

export namespace StreamEventsRequest {
  export type AsObject = {
    category: string,
    severity: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListAlertsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAlertsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAlertsRequest): ListAlertsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListAlertsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAlertsRequest;
  static deserializeBinaryFromReader(message: ListAlertsRequest, reader: jspb.BinaryReader): ListAlertsRequest;
}

export namespace ListAlertsRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
    status: string,
    severity: string,
  }
}

export class ListAlertsResponse extends jspb.Message {
  clearAlertsList(): void;
  getAlertsList(): Array<Alert>;
  setAlertsList(value: Array<Alert>): void;
  addAlerts(value?: Alert, index?: number): Alert;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAlertsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAlertsResponse): ListAlertsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListAlertsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAlertsResponse;
  static deserializeBinaryFromReader(message: ListAlertsResponse, reader: jspb.BinaryReader): ListAlertsResponse;
}

export namespace ListAlertsResponse {
  export type AsObject = {
    alertsList: Array<Alert.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class GetAlertSummaryRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAlertSummaryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAlertSummaryRequest): GetAlertSummaryRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetAlertSummaryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAlertSummaryRequest;
  static deserializeBinaryFromReader(message: GetAlertSummaryRequest, reader: jspb.BinaryReader): GetAlertSummaryRequest;
}

export namespace GetAlertSummaryRequest {
  export type AsObject = {
  }
}

export class GetAlertSummaryResponse extends jspb.Message {
  hasSummary(): boolean;
  clearSummary(): void;
  getSummary(): AlertSummary | undefined;
  setSummary(value?: AlertSummary): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAlertSummaryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetAlertSummaryResponse): GetAlertSummaryResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetAlertSummaryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAlertSummaryResponse;
  static deserializeBinaryFromReader(message: GetAlertSummaryResponse, reader: jspb.BinaryReader): GetAlertSummaryResponse;
}

export namespace GetAlertSummaryResponse {
  export type AsObject = {
    summary?: AlertSummary.AsObject,
  }
}

export class Event extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEventType(): string;
  setEventType(value: string): void;

  getCategory(): string;
  setCategory(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getSourceName(): string;
  setSourceName(value: string): void;

  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Event.AsObject;
  static toObject(includeInstance: boolean, msg: Event): Event.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Event, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Event;
  static deserializeBinaryFromReader(message: Event, reader: jspb.BinaryReader): Event;
}

export namespace Event {
  export type AsObject = {
    id: string,
    eventType: string,
    category: string,
    severity: string,
    title: string,
    description: string,
    sourceName: string,
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    data: Uint8Array | string,
  }
}

export class Alert extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  getCategory(): string;
  setCategory(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getSourceName(): string;
  setSourceName(value: string): void;

  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getStatus(): string;
  setStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Alert.AsObject;
  static toObject(includeInstance: boolean, msg: Alert): Alert.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Alert, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Alert;
  static deserializeBinaryFromReader(message: Alert, reader: jspb.BinaryReader): Alert;
}

export namespace Alert {
  export type AsObject = {
    id: string,
    severity: string,
    category: string,
    title: string,
    description: string,
    sourceName: string,
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    status: string,
  }
}

export class AlertSummary extends jspb.Message {
  getCritical(): number;
  setCritical(value: number): void;

  getWarning(): number;
  setWarning(value: number): void;

  getInfo(): number;
  setInfo(value: number): void;

  clearRecentList(): void;
  getRecentList(): Array<Alert>;
  setRecentList(value: Array<Alert>): void;
  addRecent(value?: Alert, index?: number): Alert;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AlertSummary.AsObject;
  static toObject(includeInstance: boolean, msg: AlertSummary): AlertSummary.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AlertSummary, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AlertSummary;
  static deserializeBinaryFromReader(message: AlertSummary, reader: jspb.BinaryReader): AlertSummary;
}

export namespace AlertSummary {
  export type AsObject = {
    critical: number,
    warning: number,
    info: number,
    recentList: Array<Alert.AsObject>,
  }
}

export class ListScaciSessionsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getCanResume(): boolean;
  setCanResume(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciSessionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciSessionsRequest): ListScaciSessionsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciSessionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciSessionsRequest;
  static deserializeBinaryFromReader(message: ListScaciSessionsRequest, reader: jspb.BinaryReader): ListScaciSessionsRequest;
}

export namespace ListScaciSessionsRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
    status: string,
    canResume: boolean,
  }
}

export class ListScaciSessionsResponse extends jspb.Message {
  clearSessionsList(): void;
  getSessionsList(): Array<ScaciSession>;
  setSessionsList(value: Array<ScaciSession>): void;
  addSessions(value?: ScaciSession, index?: number): ScaciSession;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciSessionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciSessionsResponse): ListScaciSessionsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciSessionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciSessionsResponse;
  static deserializeBinaryFromReader(message: ListScaciSessionsResponse, reader: jspb.BinaryReader): ListScaciSessionsResponse;
}

export namespace ListScaciSessionsResponse {
  export type AsObject = {
    sessionsList: Array<ScaciSession.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class GetScaciSessionRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciSessionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciSessionRequest): GetScaciSessionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciSessionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciSessionRequest;
  static deserializeBinaryFromReader(message: GetScaciSessionRequest, reader: jspb.BinaryReader): GetScaciSessionRequest;
}

export namespace GetScaciSessionRequest {
  export type AsObject = {
    id: string,
  }
}

export class GetScaciSessionResponse extends jspb.Message {
  hasSession(): boolean;
  clearSession(): void;
  getSession(): ScaciSession | undefined;
  setSession(value?: ScaciSession): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciSessionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciSessionResponse): GetScaciSessionResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciSessionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciSessionResponse;
  static deserializeBinaryFromReader(message: GetScaciSessionResponse, reader: jspb.BinaryReader): GetScaciSessionResponse;
}

export namespace GetScaciSessionResponse {
  export type AsObject = {
    session?: ScaciSession.AsObject,
  }
}

export class GetScaciStatisticsRequest extends jspb.Message {
  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciStatisticsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciStatisticsRequest): GetScaciStatisticsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciStatisticsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciStatisticsRequest;
  static deserializeBinaryFromReader(message: GetScaciStatisticsRequest, reader: jspb.BinaryReader): GetScaciStatisticsRequest;
}

export namespace GetScaciStatisticsRequest {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetScaciStatisticsResponse extends jspb.Message {
  hasStatistics(): boolean;
  clearStatistics(): void;
  getStatistics(): ScaciStatistics | undefined;
  setStatistics(value?: ScaciStatistics): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciStatisticsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciStatisticsResponse): GetScaciStatisticsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciStatisticsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciStatisticsResponse;
  static deserializeBinaryFromReader(message: GetScaciStatisticsResponse, reader: jspb.BinaryReader): GetScaciStatisticsResponse;
}

export namespace GetScaciStatisticsResponse {
  export type AsObject = {
    statistics?: ScaciStatistics.AsObject,
  }
}

export class ListScaciErrorsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciErrorsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciErrorsRequest): ListScaciErrorsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciErrorsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciErrorsRequest;
  static deserializeBinaryFromReader(message: ListScaciErrorsRequest, reader: jspb.BinaryReader): ListScaciErrorsRequest;
}

export namespace ListScaciErrorsRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListScaciErrorsResponse extends jspb.Message {
  clearErrorsList(): void;
  getErrorsList(): Array<ScaciError>;
  setErrorsList(value: Array<ScaciError>): void;
  addErrors(value?: ScaciError, index?: number): ScaciError;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciErrorsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciErrorsResponse): ListScaciErrorsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciErrorsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciErrorsResponse;
  static deserializeBinaryFromReader(message: ListScaciErrorsResponse, reader: jspb.BinaryReader): ListScaciErrorsResponse;
}

export namespace ListScaciErrorsResponse {
  export type AsObject = {
    errorsList: Array<ScaciError.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class ListScaciQueuesRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciQueuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciQueuesRequest): ListScaciQueuesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciQueuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciQueuesRequest;
  static deserializeBinaryFromReader(message: ListScaciQueuesRequest, reader: jspb.BinaryReader): ListScaciQueuesRequest;
}

export namespace ListScaciQueuesRequest {
  export type AsObject = {
    epEui: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListScaciQueuesResponse extends jspb.Message {
  clearQueueEntriesList(): void;
  getQueueEntriesList(): Array<ScaciQueueEntry>;
  setQueueEntriesList(value: Array<ScaciQueueEntry>): void;
  addQueueEntries(value?: ScaciQueueEntry, index?: number): ScaciQueueEntry;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListScaciQueuesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListScaciQueuesResponse): ListScaciQueuesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListScaciQueuesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListScaciQueuesResponse;
  static deserializeBinaryFromReader(message: ListScaciQueuesResponse, reader: jspb.BinaryReader): ListScaciQueuesResponse;
}

export namespace ListScaciQueuesResponse {
  export type AsObject = {
    queueEntriesList: Array<ScaciQueueEntry.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class GetScaciStatusRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciStatusRequest): GetScaciStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciStatusRequest;
  static deserializeBinaryFromReader(message: GetScaciStatusRequest, reader: jspb.BinaryReader): GetScaciStatusRequest;
}

export namespace GetScaciStatusRequest {
  export type AsObject = {
  }
}

export class GetScaciStatusResponse extends jspb.Message {
  hasStatus(): boolean;
  clearStatus(): void;
  getStatus(): ScaciStatus | undefined;
  setStatus(value?: ScaciStatus): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetScaciStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetScaciStatusResponse): GetScaciStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetScaciStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetScaciStatusResponse;
  static deserializeBinaryFromReader(message: GetScaciStatusResponse, reader: jspb.BinaryReader): GetScaciStatusResponse;
}

export namespace GetScaciStatusResponse {
  export type AsObject = {
    status?: ScaciStatus.AsObject,
  }
}

export class ScaciSession extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getAcEui(): string;
  setAcEui(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getCanResume(): boolean;
  setCanResume(value: boolean): void;

  getProtocolVersion(): string;
  setProtocolVersion(value: string): void;

  hasConnectedAt(): boolean;
  clearConnectedAt(): void;
  getConnectedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setConnectedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastActivityAt(): boolean;
  clearLastActivityAt(): void;
  getLastActivityAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastActivityAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getOperationsCount(): number;
  setOperationsCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScaciSession.AsObject;
  static toObject(includeInstance: boolean, msg: ScaciSession): ScaciSession.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScaciSession, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScaciSession;
  static deserializeBinaryFromReader(message: ScaciSession, reader: jspb.BinaryReader): ScaciSession;
}

export namespace ScaciSession {
  export type AsObject = {
    id: string,
    acEui: string,
    status: string,
    canResume: boolean,
    protocolVersion: string,
    connectedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastActivityAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    operationsCount: number,
  }
}

export class ScaciStatistics extends jspb.Message {
  getTotalSessions(): number;
  setTotalSessions(value: number): void;

  getActiveSessions(): number;
  setActiveSessions(value: number): void;

  getTotalOperations(): number;
  setTotalOperations(value: number): void;

  getSuccessfulOperations(): number;
  setSuccessfulOperations(value: number): void;

  getFailedOperations(): number;
  setFailedOperations(value: number): void;

  getSuccessRate(): number;
  setSuccessRate(value: number): void;

  hasUptimeSince(): boolean;
  clearUptimeSince(): void;
  getUptimeSince(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUptimeSince(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScaciStatistics.AsObject;
  static toObject(includeInstance: boolean, msg: ScaciStatistics): ScaciStatistics.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScaciStatistics, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScaciStatistics;
  static deserializeBinaryFromReader(message: ScaciStatistics, reader: jspb.BinaryReader): ScaciStatistics;
}

export namespace ScaciStatistics {
  export type AsObject = {
    totalSessions: number,
    activeSessions: number,
    totalOperations: number,
    successfulOperations: number,
    failedOperations: number,
    successRate: number,
    uptimeSince?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ScaciError extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getErrorCode(): string;
  setErrorCode(value: string): void;

  getErrorMessage(): string;
  setErrorMessage(value: string): void;

  getSessionId(): string;
  setSessionId(value: string): void;

  getOperationType(): string;
  setOperationType(value: string): void;

  hasOccurredAt(): boolean;
  clearOccurredAt(): void;
  getOccurredAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setOccurredAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScaciError.AsObject;
  static toObject(includeInstance: boolean, msg: ScaciError): ScaciError.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScaciError, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScaciError;
  static deserializeBinaryFromReader(message: ScaciError, reader: jspb.BinaryReader): ScaciError;
}

export namespace ScaciError {
  export type AsObject = {
    id: string,
    errorCode: string,
    errorMessage: string,
    sessionId: string,
    operationType: string,
    occurredAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ScaciQueueEntry extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  getOperationType(): string;
  setOperationType(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): void;

  hasQueuedAt(): boolean;
  clearQueuedAt(): void;
  getQueuedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setQueuedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasProcessedAt(): boolean;
  clearProcessedAt(): void;
  getProcessedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setProcessedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScaciQueueEntry.AsObject;
  static toObject(includeInstance: boolean, msg: ScaciQueueEntry): ScaciQueueEntry.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScaciQueueEntry, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScaciQueueEntry;
  static deserializeBinaryFromReader(message: ScaciQueueEntry, reader: jspb.BinaryReader): ScaciQueueEntry;
}

export namespace ScaciQueueEntry {
  export type AsObject = {
    id: string,
    epEui: string,
    operationType: string,
    status: string,
    payload: Uint8Array | string,
    queuedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    processedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ScaciStatus extends jspb.Message {
  getServiceOnline(): boolean;
  setServiceOnline(value: boolean): void;

  getActiveSessions(): number;
  setActiveSessions(value: number): void;

  getPendingOperations(): number;
  setPendingOperations(value: number): void;

  hasUptimeSince(): boolean;
  clearUptimeSince(): void;
  getUptimeSince(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUptimeSince(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getProtocolVersion(): string;
  setProtocolVersion(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScaciStatus.AsObject;
  static toObject(includeInstance: boolean, msg: ScaciStatus): ScaciStatus.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ScaciStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScaciStatus;
  static deserializeBinaryFromReader(message: ScaciStatus, reader: jspb.BinaryReader): ScaciStatus;
}

export namespace ScaciStatus {
  export type AsObject = {
    serviceOnline: boolean,
    activeSessions: number,
    pendingOperations: number,
    uptimeSince?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    protocolVersion: string,
  }
}

export class GenerateCertificateRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getBaseStationName(): string;
  setBaseStationName(value: string): void;

  getValidityDays(): number;
  setValidityDays(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateCertificateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateCertificateRequest): GenerateCertificateRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateCertificateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateCertificateRequest;
  static deserializeBinaryFromReader(message: GenerateCertificateRequest, reader: jspb.BinaryReader): GenerateCertificateRequest;
}

export namespace GenerateCertificateRequest {
  export type AsObject = {
    bsEui: string,
    baseStationName: string,
    validityDays: number,
  }
}

export class GenerateCertificateResponse extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getServiceCenterUrl(): string;
  setServiceCenterUrl(value: string): void;

  getDownloadUrlsMap(): jspb.Map<string, string>;
  clearDownloadUrlsMap(): void;
  hasExpiresAt(): boolean;
  clearExpiresAt(): void;
  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateCertificateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateCertificateResponse): GenerateCertificateResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateCertificateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateCertificateResponse;
  static deserializeBinaryFromReader(message: GenerateCertificateResponse, reader: jspb.BinaryReader): GenerateCertificateResponse;
}

export namespace GenerateCertificateResponse {
  export type AsObject = {
    bsEui: string,
    serviceCenterUrl: string,
    downloadUrlsMap: Array<[string, string]>,
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class DownloadCertificateRequest extends jspb.Message {
  getCertType(): string;
  setCertType(value: string): void;

  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownloadCertificateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DownloadCertificateRequest): DownloadCertificateRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DownloadCertificateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownloadCertificateRequest;
  static deserializeBinaryFromReader(message: DownloadCertificateRequest, reader: jspb.BinaryReader): DownloadCertificateRequest;
}

export namespace DownloadCertificateRequest {
  export type AsObject = {
    certType: string,
    id: string,
  }
}

export class DownloadCertificateResponse extends jspb.Message {
  getContent(): Uint8Array | string;
  getContent_asU8(): Uint8Array;
  getContent_asB64(): string;
  setContent(value: Uint8Array | string): void;

  getFilename(): string;
  setFilename(value: string): void;

  getContentType(): string;
  setContentType(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownloadCertificateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DownloadCertificateResponse): DownloadCertificateResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DownloadCertificateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownloadCertificateResponse;
  static deserializeBinaryFromReader(message: DownloadCertificateResponse, reader: jspb.BinaryReader): DownloadCertificateResponse;
}

export namespace DownloadCertificateResponse {
  export type AsObject = {
    content: Uint8Array | string,
    filename: string,
    contentType: string,
  }
}

export class DownloadBaseStationCertificateRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getCertType(): string;
  setCertType(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownloadBaseStationCertificateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DownloadBaseStationCertificateRequest): DownloadBaseStationCertificateRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DownloadBaseStationCertificateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownloadBaseStationCertificateRequest;
  static deserializeBinaryFromReader(message: DownloadBaseStationCertificateRequest, reader: jspb.BinaryReader): DownloadBaseStationCertificateRequest;
}

export namespace DownloadBaseStationCertificateRequest {
  export type AsObject = {
    bsEui: string,
    certType: string,
  }
}

export class GenerateServerCertificatesRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateServerCertificatesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateServerCertificatesRequest): GenerateServerCertificatesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateServerCertificatesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateServerCertificatesRequest;
  static deserializeBinaryFromReader(message: GenerateServerCertificatesRequest, reader: jspb.BinaryReader): GenerateServerCertificatesRequest;
}

export namespace GenerateServerCertificatesRequest {
  export type AsObject = {
  }
}

export class GenerateServerCertificatesResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getMessage(): string;
  setMessage(value: string): void;

  hasExpiresAt(): boolean;
  clearExpiresAt(): void;
  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateServerCertificatesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateServerCertificatesResponse): GenerateServerCertificatesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GenerateServerCertificatesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateServerCertificatesResponse;
  static deserializeBinaryFromReader(message: GenerateServerCertificatesResponse, reader: jspb.BinaryReader): GenerateServerCertificatesResponse;
}

export namespace GenerateServerCertificatesResponse {
  export type AsObject = {
    success: boolean,
    message: string,
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class RenewServerCertificatesRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RenewServerCertificatesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RenewServerCertificatesRequest): RenewServerCertificatesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RenewServerCertificatesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RenewServerCertificatesRequest;
  static deserializeBinaryFromReader(message: RenewServerCertificatesRequest, reader: jspb.BinaryReader): RenewServerCertificatesRequest;
}

export namespace RenewServerCertificatesRequest {
  export type AsObject = {
  }
}

export class RenewServerCertificatesResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getMessage(): string;
  setMessage(value: string): void;

  hasExpiresAt(): boolean;
  clearExpiresAt(): void;
  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RenewServerCertificatesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RenewServerCertificatesResponse): RenewServerCertificatesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RenewServerCertificatesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RenewServerCertificatesResponse;
  static deserializeBinaryFromReader(message: RenewServerCertificatesResponse, reader: jspb.BinaryReader): RenewServerCertificatesResponse;
}

export namespace RenewServerCertificatesResponse {
  export type AsObject = {
    success: boolean,
    message: string,
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetServerCertificateStatusRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetServerCertificateStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetServerCertificateStatusRequest): GetServerCertificateStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetServerCertificateStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetServerCertificateStatusRequest;
  static deserializeBinaryFromReader(message: GetServerCertificateStatusRequest, reader: jspb.BinaryReader): GetServerCertificateStatusRequest;
}

export namespace GetServerCertificateStatusRequest {
  export type AsObject = {
  }
}

export class GetServerCertificateStatusResponse extends jspb.Message {
  hasServerCert(): boolean;
  clearServerCert(): void;
  getServerCert(): CertificateStatus | undefined;
  setServerCert(value?: CertificateStatus): void;

  hasCaCert(): boolean;
  clearCaCert(): void;
  getCaCert(): CertificateStatus | undefined;
  setCaCert(value?: CertificateStatus): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetServerCertificateStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetServerCertificateStatusResponse): GetServerCertificateStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetServerCertificateStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetServerCertificateStatusResponse;
  static deserializeBinaryFromReader(message: GetServerCertificateStatusResponse, reader: jspb.BinaryReader): GetServerCertificateStatusResponse;
}

export namespace GetServerCertificateStatusResponse {
  export type AsObject = {
    serverCert?: CertificateStatus.AsObject,
    caCert?: CertificateStatus.AsObject,
  }
}

export class CertificateStatus extends jspb.Message {
  getSubject(): string;
  setSubject(value: string): void;

  getIssuer(): string;
  setIssuer(value: string): void;

  hasNotBefore(): boolean;
  clearNotBefore(): void;
  getNotBefore(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setNotBefore(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasNotAfter(): boolean;
  clearNotAfter(): void;
  getNotAfter(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setNotAfter(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDaysUntilExpiry(): number;
  setDaysUntilExpiry(value: number): void;

  getIsValid(): boolean;
  setIsValid(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CertificateStatus.AsObject;
  static toObject(includeInstance: boolean, msg: CertificateStatus): CertificateStatus.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CertificateStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CertificateStatus;
  static deserializeBinaryFromReader(message: CertificateStatus, reader: jspb.BinaryReader): CertificateStatus;
}

export namespace CertificateStatus {
  export type AsObject = {
    subject: string,
    issuer: string,
    notBefore?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    notAfter?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    daysUntilExpiry: number,
    isValid: boolean,
  }
}

export class CreateManufacturerRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getCode(): string;
  setCode(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getWebsite(): string;
  setWebsite(value: string): void;

  getContactEmail(): string;
  setContactEmail(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateManufacturerRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateManufacturerRequest): CreateManufacturerRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateManufacturerRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateManufacturerRequest;
  static deserializeBinaryFromReader(message: CreateManufacturerRequest, reader: jspb.BinaryReader): CreateManufacturerRequest;
}

export namespace CreateManufacturerRequest {
  export type AsObject = {
    name: string,
    code: string,
    description: string,
    website: string,
    contactEmail: string,
  }
}

export class CreateManufacturerResponse extends jspb.Message {
  hasManufacturer(): boolean;
  clearManufacturer(): void;
  getManufacturer(): Manufacturer | undefined;
  setManufacturer(value?: Manufacturer): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateManufacturerResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CreateManufacturerResponse): CreateManufacturerResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateManufacturerResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateManufacturerResponse;
  static deserializeBinaryFromReader(message: CreateManufacturerResponse, reader: jspb.BinaryReader): CreateManufacturerResponse;
}

export namespace CreateManufacturerResponse {
  export type AsObject = {
    manufacturer?: Manufacturer.AsObject,
  }
}

export class GetManufacturerRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetManufacturerRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetManufacturerRequest): GetManufacturerRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetManufacturerRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetManufacturerRequest;
  static deserializeBinaryFromReader(message: GetManufacturerRequest, reader: jspb.BinaryReader): GetManufacturerRequest;
}

export namespace GetManufacturerRequest {
  export type AsObject = {
    id: string,
  }
}

export class GetManufacturerResponse extends jspb.Message {
  hasManufacturer(): boolean;
  clearManufacturer(): void;
  getManufacturer(): Manufacturer | undefined;
  setManufacturer(value?: Manufacturer): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetManufacturerResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetManufacturerResponse): GetManufacturerResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetManufacturerResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetManufacturerResponse;
  static deserializeBinaryFromReader(message: GetManufacturerResponse, reader: jspb.BinaryReader): GetManufacturerResponse;
}

export namespace GetManufacturerResponse {
  export type AsObject = {
    manufacturer?: Manufacturer.AsObject,
  }
}

export class UpdateManufacturerRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getWebsite(): string;
  setWebsite(value: string): void;

  getContactEmail(): string;
  setContactEmail(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateManufacturerRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateManufacturerRequest): UpdateManufacturerRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateManufacturerRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateManufacturerRequest;
  static deserializeBinaryFromReader(message: UpdateManufacturerRequest, reader: jspb.BinaryReader): UpdateManufacturerRequest;
}

export namespace UpdateManufacturerRequest {
  export type AsObject = {
    id: string,
    name: string,
    description: string,
    website: string,
    contactEmail: string,
  }
}

export class UpdateManufacturerResponse extends jspb.Message {
  hasManufacturer(): boolean;
  clearManufacturer(): void;
  getManufacturer(): Manufacturer | undefined;
  setManufacturer(value?: Manufacturer): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateManufacturerResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateManufacturerResponse): UpdateManufacturerResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateManufacturerResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateManufacturerResponse;
  static deserializeBinaryFromReader(message: UpdateManufacturerResponse, reader: jspb.BinaryReader): UpdateManufacturerResponse;
}

export namespace UpdateManufacturerResponse {
  export type AsObject = {
    manufacturer?: Manufacturer.AsObject,
  }
}

export class DeleteManufacturerRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteManufacturerRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteManufacturerRequest): DeleteManufacturerRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteManufacturerRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteManufacturerRequest;
  static deserializeBinaryFromReader(message: DeleteManufacturerRequest, reader: jspb.BinaryReader): DeleteManufacturerRequest;
}

export namespace DeleteManufacturerRequest {
  export type AsObject = {
    id: string,
  }
}

export class DeleteManufacturerResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteManufacturerResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteManufacturerResponse): DeleteManufacturerResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteManufacturerResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteManufacturerResponse;
  static deserializeBinaryFromReader(message: DeleteManufacturerResponse, reader: jspb.BinaryReader): DeleteManufacturerResponse;
}

export namespace DeleteManufacturerResponse {
  export type AsObject = {
    success: boolean,
  }
}

export class ListManufacturersRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListManufacturersRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListManufacturersRequest): ListManufacturersRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListManufacturersRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListManufacturersRequest;
  static deserializeBinaryFromReader(message: ListManufacturersRequest, reader: jspb.BinaryReader): ListManufacturersRequest;
}

export namespace ListManufacturersRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
  }
}

export class ListManufacturersResponse extends jspb.Message {
  clearManufacturersList(): void;
  getManufacturersList(): Array<Manufacturer>;
  setManufacturersList(value: Array<Manufacturer>): void;
  addManufacturers(value?: Manufacturer, index?: number): Manufacturer;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListManufacturersResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListManufacturersResponse): ListManufacturersResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListManufacturersResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListManufacturersResponse;
  static deserializeBinaryFromReader(message: ListManufacturersResponse, reader: jspb.BinaryReader): ListManufacturersResponse;
}

export namespace ListManufacturersResponse {
  export type AsObject = {
    manufacturersList: Array<Manufacturer.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class Manufacturer extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getCode(): string;
  setCode(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getWebsite(): string;
  setWebsite(value: string): void;

  getContactEmail(): string;
  setContactEmail(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getIsVerified(): boolean;
  setIsVerified(value: boolean): void;

  getModelCount(): number;
  setModelCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Manufacturer.AsObject;
  static toObject(includeInstance: boolean, msg: Manufacturer): Manufacturer.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Manufacturer, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Manufacturer;
  static deserializeBinaryFromReader(message: Manufacturer, reader: jspb.BinaryReader): Manufacturer;
}

export namespace Manufacturer {
  export type AsObject = {
    id: string,
    name: string,
    code: string,
    description: string,
    website: string,
    contactEmail: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    tenantId: string,
    isVerified: boolean,
    modelCount: number,
  }
}

export class CreateDeviceModelRequest extends jspb.Message {
  getManufacturerId(): string;
  setManufacturerId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getCode(): string;
  setCode(value: string): void;

  getTypeEui(): string;
  setTypeEui(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateDeviceModelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateDeviceModelRequest): CreateDeviceModelRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateDeviceModelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateDeviceModelRequest;
  static deserializeBinaryFromReader(message: CreateDeviceModelRequest, reader: jspb.BinaryReader): CreateDeviceModelRequest;
}

export namespace CreateDeviceModelRequest {
  export type AsObject = {
    manufacturerId: string,
    name: string,
    code: string,
    typeEui: string,
    description: string,
  }
}

export class CreateDeviceModelResponse extends jspb.Message {
  hasDeviceModel(): boolean;
  clearDeviceModel(): void;
  getDeviceModel(): DeviceModel | undefined;
  setDeviceModel(value?: DeviceModel): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateDeviceModelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CreateDeviceModelResponse): CreateDeviceModelResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateDeviceModelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateDeviceModelResponse;
  static deserializeBinaryFromReader(message: CreateDeviceModelResponse, reader: jspb.BinaryReader): CreateDeviceModelResponse;
}

export namespace CreateDeviceModelResponse {
  export type AsObject = {
    deviceModel?: DeviceModel.AsObject,
  }
}

export class GetDeviceModelRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDeviceModelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDeviceModelRequest): GetDeviceModelRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDeviceModelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDeviceModelRequest;
  static deserializeBinaryFromReader(message: GetDeviceModelRequest, reader: jspb.BinaryReader): GetDeviceModelRequest;
}

export namespace GetDeviceModelRequest {
  export type AsObject = {
    id: string,
  }
}

export class GetDeviceModelResponse extends jspb.Message {
  hasDeviceModel(): boolean;
  clearDeviceModel(): void;
  getDeviceModel(): DeviceModel | undefined;
  setDeviceModel(value?: DeviceModel): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDeviceModelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetDeviceModelResponse): GetDeviceModelResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetDeviceModelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDeviceModelResponse;
  static deserializeBinaryFromReader(message: GetDeviceModelResponse, reader: jspb.BinaryReader): GetDeviceModelResponse;
}

export namespace GetDeviceModelResponse {
  export type AsObject = {
    deviceModel?: DeviceModel.AsObject,
  }
}

export class UpdateDeviceModelRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateDeviceModelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateDeviceModelRequest): UpdateDeviceModelRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateDeviceModelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateDeviceModelRequest;
  static deserializeBinaryFromReader(message: UpdateDeviceModelRequest, reader: jspb.BinaryReader): UpdateDeviceModelRequest;
}

export namespace UpdateDeviceModelRequest {
  export type AsObject = {
    id: string,
    name: string,
    description: string,
  }
}

export class UpdateDeviceModelResponse extends jspb.Message {
  hasDeviceModel(): boolean;
  clearDeviceModel(): void;
  getDeviceModel(): DeviceModel | undefined;
  setDeviceModel(value?: DeviceModel): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateDeviceModelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateDeviceModelResponse): UpdateDeviceModelResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateDeviceModelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateDeviceModelResponse;
  static deserializeBinaryFromReader(message: UpdateDeviceModelResponse, reader: jspb.BinaryReader): UpdateDeviceModelResponse;
}

export namespace UpdateDeviceModelResponse {
  export type AsObject = {
    deviceModel?: DeviceModel.AsObject,
  }
}

export class DeleteDeviceModelRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteDeviceModelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteDeviceModelRequest): DeleteDeviceModelRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteDeviceModelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteDeviceModelRequest;
  static deserializeBinaryFromReader(message: DeleteDeviceModelRequest, reader: jspb.BinaryReader): DeleteDeviceModelRequest;
}

export namespace DeleteDeviceModelRequest {
  export type AsObject = {
    id: string,
  }
}

export class DeleteDeviceModelResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteDeviceModelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteDeviceModelResponse): DeleteDeviceModelResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteDeviceModelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteDeviceModelResponse;
  static deserializeBinaryFromReader(message: DeleteDeviceModelResponse, reader: jspb.BinaryReader): DeleteDeviceModelResponse;
}

export namespace DeleteDeviceModelResponse {
  export type AsObject = {
    success: boolean,
  }
}

export class ListDeviceModelsRequest extends jspb.Message {
  getManufacturerId(): string;
  setManufacturerId(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDeviceModelsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListDeviceModelsRequest): ListDeviceModelsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDeviceModelsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDeviceModelsRequest;
  static deserializeBinaryFromReader(message: ListDeviceModelsRequest, reader: jspb.BinaryReader): ListDeviceModelsRequest;
}

export namespace ListDeviceModelsRequest {
  export type AsObject = {
    manufacturerId: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListDeviceModelsResponse extends jspb.Message {
  clearDeviceModelsList(): void;
  getDeviceModelsList(): Array<DeviceModel>;
  setDeviceModelsList(value: Array<DeviceModel>): void;
  addDeviceModels(value?: DeviceModel, index?: number): DeviceModel;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDeviceModelsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListDeviceModelsResponse): ListDeviceModelsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListDeviceModelsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDeviceModelsResponse;
  static deserializeBinaryFromReader(message: ListDeviceModelsResponse, reader: jspb.BinaryReader): ListDeviceModelsResponse;
}

export namespace ListDeviceModelsResponse {
  export type AsObject = {
    deviceModelsList: Array<DeviceModel.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class DeviceModel extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getManufacturerId(): string;
  setManufacturerId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getCode(): string;
  setCode(value: string): void;

  getTypeEui(): string;
  setTypeEui(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getTenantId(): string;
  setTenantId(value: string): void;

  getDatasheetUrl(): string;
  setDatasheetUrl(value: string): void;

  getBlueprintCount(): number;
  setBlueprintCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceModel.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceModel): DeviceModel.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceModel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceModel;
  static deserializeBinaryFromReader(message: DeviceModel, reader: jspb.BinaryReader): DeviceModel;
}

export namespace DeviceModel {
  export type AsObject = {
    id: string,
    manufacturerId: string,
    name: string,
    code: string,
    typeEui: string,
    description: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    tenantId: string,
    datasheetUrl: string,
    blueprintCount: number,
  }
}

export class CreateBlueprintRequest extends jspb.Message {
  getDeviceModelId(): string;
  setDeviceModelId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getVersion(): string;
  setVersion(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getDecoderScript(): Uint8Array | string;
  getDecoderScript_asU8(): Uint8Array;
  getDecoderScript_asB64(): string;
  setDecoderScript(value: Uint8Array | string): void;

  getIsDefault(): boolean;
  setIsDefault(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateBlueprintRequest): CreateBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateBlueprintRequest;
  static deserializeBinaryFromReader(message: CreateBlueprintRequest, reader: jspb.BinaryReader): CreateBlueprintRequest;
}

export namespace CreateBlueprintRequest {
  export type AsObject = {
    deviceModelId: string,
    name: string,
    version: string,
    description: string,
    decoderScript: Uint8Array | string,
    isDefault: boolean,
  }
}

export class CreateBlueprintResponse extends jspb.Message {
  hasBlueprint(): boolean;
  clearBlueprint(): void;
  getBlueprint(): Blueprint | undefined;
  setBlueprint(value?: Blueprint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CreateBlueprintResponse): CreateBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateBlueprintResponse;
  static deserializeBinaryFromReader(message: CreateBlueprintResponse, reader: jspb.BinaryReader): CreateBlueprintResponse;
}

export namespace CreateBlueprintResponse {
  export type AsObject = {
    blueprint?: Blueprint.AsObject,
  }
}

export class GetBlueprintRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBlueprintRequest): GetBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBlueprintRequest;
  static deserializeBinaryFromReader(message: GetBlueprintRequest, reader: jspb.BinaryReader): GetBlueprintRequest;
}

export namespace GetBlueprintRequest {
  export type AsObject = {
    id: string,
  }
}

export class GetBlueprintResponse extends jspb.Message {
  hasBlueprint(): boolean;
  clearBlueprint(): void;
  getBlueprint(): Blueprint | undefined;
  setBlueprint(value?: Blueprint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBlueprintResponse): GetBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBlueprintResponse;
  static deserializeBinaryFromReader(message: GetBlueprintResponse, reader: jspb.BinaryReader): GetBlueprintResponse;
}

export namespace GetBlueprintResponse {
  export type AsObject = {
    blueprint?: Blueprint.AsObject,
  }
}

export class UpdateBlueprintRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getVersion(): string;
  setVersion(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getDecoderScript(): Uint8Array | string;
  getDecoderScript_asU8(): Uint8Array;
  getDecoderScript_asB64(): string;
  setDecoderScript(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBlueprintRequest): UpdateBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBlueprintRequest;
  static deserializeBinaryFromReader(message: UpdateBlueprintRequest, reader: jspb.BinaryReader): UpdateBlueprintRequest;
}

export namespace UpdateBlueprintRequest {
  export type AsObject = {
    id: string,
    name: string,
    version: string,
    description: string,
    decoderScript: Uint8Array | string,
  }
}

export class UpdateBlueprintResponse extends jspb.Message {
  hasBlueprint(): boolean;
  clearBlueprint(): void;
  getBlueprint(): Blueprint | undefined;
  setBlueprint(value?: Blueprint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBlueprintResponse): UpdateBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpdateBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBlueprintResponse;
  static deserializeBinaryFromReader(message: UpdateBlueprintResponse, reader: jspb.BinaryReader): UpdateBlueprintResponse;
}

export namespace UpdateBlueprintResponse {
  export type AsObject = {
    blueprint?: Blueprint.AsObject,
  }
}

export class DeleteBlueprintRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteBlueprintRequest): DeleteBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteBlueprintRequest;
  static deserializeBinaryFromReader(message: DeleteBlueprintRequest, reader: jspb.BinaryReader): DeleteBlueprintRequest;
}

export namespace DeleteBlueprintRequest {
  export type AsObject = {
    id: string,
  }
}

export class DeleteBlueprintResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteBlueprintResponse): DeleteBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeleteBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteBlueprintResponse;
  static deserializeBinaryFromReader(message: DeleteBlueprintResponse, reader: jspb.BinaryReader): DeleteBlueprintResponse;
}

export namespace DeleteBlueprintResponse {
  export type AsObject = {
    success: boolean,
  }
}

export class ListBlueprintsRequest extends jspb.Message {
  getDeviceModelId(): string;
  setDeviceModelId(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBlueprintsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBlueprintsRequest): ListBlueprintsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBlueprintsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBlueprintsRequest;
  static deserializeBinaryFromReader(message: ListBlueprintsRequest, reader: jspb.BinaryReader): ListBlueprintsRequest;
}

export namespace ListBlueprintsRequest {
  export type AsObject = {
    deviceModelId: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListBlueprintsResponse extends jspb.Message {
  clearBlueprintsList(): void;
  getBlueprintsList(): Array<Blueprint>;
  setBlueprintsList(value: Array<Blueprint>): void;
  addBlueprints(value?: Blueprint, index?: number): Blueprint;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBlueprintsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBlueprintsResponse): ListBlueprintsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBlueprintsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBlueprintsResponse;
  static deserializeBinaryFromReader(message: ListBlueprintsResponse, reader: jspb.BinaryReader): ListBlueprintsResponse;
}

export namespace ListBlueprintsResponse {
  export type AsObject = {
    blueprintsList: Array<Blueprint.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class SetDefaultBlueprintRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDefaultBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SetDefaultBlueprintRequest): SetDefaultBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SetDefaultBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDefaultBlueprintRequest;
  static deserializeBinaryFromReader(message: SetDefaultBlueprintRequest, reader: jspb.BinaryReader): SetDefaultBlueprintRequest;
}

export namespace SetDefaultBlueprintRequest {
  export type AsObject = {
    id: string,
  }
}

export class SetDefaultBlueprintResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  hasBlueprint(): boolean;
  clearBlueprint(): void;
  getBlueprint(): Blueprint | undefined;
  setBlueprint(value?: Blueprint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDefaultBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDefaultBlueprintResponse): SetDefaultBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SetDefaultBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDefaultBlueprintResponse;
  static deserializeBinaryFromReader(message: SetDefaultBlueprintResponse, reader: jspb.BinaryReader): SetDefaultBlueprintResponse;
}

export namespace SetDefaultBlueprintResponse {
  export type AsObject = {
    success: boolean,
    blueprint?: Blueprint.AsObject,
  }
}

export class SubmitBlueprintToRegistryRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getContributorName(): string;
  setContributorName(value: string): void;

  getContributorEmail(): string;
  setContributorEmail(value: string): void;

  getNotes(): string;
  setNotes(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubmitBlueprintToRegistryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SubmitBlueprintToRegistryRequest): SubmitBlueprintToRegistryRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubmitBlueprintToRegistryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubmitBlueprintToRegistryRequest;
  static deserializeBinaryFromReader(message: SubmitBlueprintToRegistryRequest, reader: jspb.BinaryReader): SubmitBlueprintToRegistryRequest;
}

export namespace SubmitBlueprintToRegistryRequest {
  export type AsObject = {
    id: string,
    contributorName: string,
    contributorEmail: string,
    notes: string,
  }
}

export class SubmitBlueprintToRegistryResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getPrUrl(): string;
  setPrUrl(value: string): void;

  getCommitSha(): string;
  setCommitSha(value: string): void;

  getBranchName(): string;
  setBranchName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubmitBlueprintToRegistryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SubmitBlueprintToRegistryResponse): SubmitBlueprintToRegistryResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubmitBlueprintToRegistryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubmitBlueprintToRegistryResponse;
  static deserializeBinaryFromReader(message: SubmitBlueprintToRegistryResponse, reader: jspb.BinaryReader): SubmitBlueprintToRegistryResponse;
}

export namespace SubmitBlueprintToRegistryResponse {
  export type AsObject = {
    success: boolean,
    prUrl: string,
    commitSha: string,
    branchName: string,
  }
}

export class Blueprint extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDeviceModelId(): string;
  setDeviceModelId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getVersion(): string;
  setVersion(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getDecoderScript(): Uint8Array | string;
  getDecoderScript_asU8(): Uint8Array;
  getDecoderScript_asB64(): string;
  setDecoderScript(value: Uint8Array | string): void;

  getIsDefault(): boolean;
  setIsDefault(value: boolean): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getTypeEui(): string;
  setTypeEui(value: string): void;

  getSpecJson(): Uint8Array | string;
  getSpecJson_asU8(): Uint8Array;
  getSpecJson_asB64(): string;
  setSpecJson(value: Uint8Array | string): void;

  getRegistryRepo(): string;
  setRegistryRepo(value: string): void;

  getRegistryCommitSha(): string;
  setRegistryCommitSha(value: string): void;

  getRegistryVerified(): boolean;
  setRegistryVerified(value: boolean): void;

  getRegistryPrUrl(): string;
  setRegistryPrUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Blueprint.AsObject;
  static toObject(includeInstance: boolean, msg: Blueprint): Blueprint.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Blueprint, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Blueprint;
  static deserializeBinaryFromReader(message: Blueprint, reader: jspb.BinaryReader): Blueprint;
}

export namespace Blueprint {
  export type AsObject = {
    id: string,
    deviceModelId: string,
    name: string,
    version: string,
    description: string,
    decoderScript: Uint8Array | string,
    isDefault: boolean,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    typeEui: string,
    specJson: Uint8Array | string,
    registryRepo: string,
    registryCommitSha: string,
    registryVerified: boolean,
    registryPrUrl: string,
  }
}

export class CreateDeviceModelWithBlueprintRequest extends jspb.Message {
  getManufacturerId(): string;
  setManufacturerId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getVersion(): string;
  setVersion(value: string): void;

  getDecoderScript(): Uint8Array | string;
  getDecoderScript_asU8(): Uint8Array;
  getDecoderScript_asB64(): string;
  setDecoderScript(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateDeviceModelWithBlueprintRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateDeviceModelWithBlueprintRequest): CreateDeviceModelWithBlueprintRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateDeviceModelWithBlueprintRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateDeviceModelWithBlueprintRequest;
  static deserializeBinaryFromReader(message: CreateDeviceModelWithBlueprintRequest, reader: jspb.BinaryReader): CreateDeviceModelWithBlueprintRequest;
}

export namespace CreateDeviceModelWithBlueprintRequest {
  export type AsObject = {
    manufacturerId: string,
    name: string,
    version: string,
    decoderScript: Uint8Array | string,
  }
}

export class CreateDeviceModelWithBlueprintResponse extends jspb.Message {
  hasDeviceModel(): boolean;
  clearDeviceModel(): void;
  getDeviceModel(): DeviceModel | undefined;
  setDeviceModel(value?: DeviceModel): void;

  hasBlueprint(): boolean;
  clearBlueprint(): void;
  getBlueprint(): Blueprint | undefined;
  setBlueprint(value?: Blueprint): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateDeviceModelWithBlueprintResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CreateDeviceModelWithBlueprintResponse): CreateDeviceModelWithBlueprintResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CreateDeviceModelWithBlueprintResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateDeviceModelWithBlueprintResponse;
  static deserializeBinaryFromReader(message: CreateDeviceModelWithBlueprintResponse, reader: jspb.BinaryReader): CreateDeviceModelWithBlueprintResponse;
}

export namespace CreateDeviceModelWithBlueprintResponse {
  export type AsObject = {
    deviceModel?: DeviceModel.AsObject,
    blueprint?: Blueprint.AsObject,
  }
}

export class DecodePreviewRequest extends jspb.Message {
  getBlueprintId(): string;
  setBlueprintId(value: string): void;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): void;

  getFormatId(): number;
  setFormatId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DecodePreviewRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DecodePreviewRequest): DecodePreviewRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DecodePreviewRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DecodePreviewRequest;
  static deserializeBinaryFromReader(message: DecodePreviewRequest, reader: jspb.BinaryReader): DecodePreviewRequest;
}

export namespace DecodePreviewRequest {
  export type AsObject = {
    blueprintId: string,
    payload: Uint8Array | string,
    formatId: number,
  }
}

export class DecodePreviewResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  getDecodedPayload(): Uint8Array | string;
  getDecodedPayload_asU8(): Uint8Array;
  getDecodedPayload_asB64(): string;
  setDecodedPayload(value: Uint8Array | string): void;

  getErrorCode(): string;
  setErrorCode(value: string): void;

  getErrorDetail(): string;
  setErrorDetail(value: string): void;

  getFormatId(): number;
  setFormatId(value: number): void;

  getBlueprintVersion(): string;
  setBlueprintVersion(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DecodePreviewResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DecodePreviewResponse): DecodePreviewResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DecodePreviewResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DecodePreviewResponse;
  static deserializeBinaryFromReader(message: DecodePreviewResponse, reader: jspb.BinaryReader): DecodePreviewResponse;
}

export namespace DecodePreviewResponse {
  export type AsObject = {
    success: boolean,
    decodedPayload: Uint8Array | string,
    errorCode: string,
    errorDetail: string,
    formatId: number,
    blueprintVersion: string,
  }
}

export class ListMessagesRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListMessagesRequest): ListMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListMessagesRequest;
  static deserializeBinaryFromReader(message: ListMessagesRequest, reader: jspb.BinaryReader): ListMessagesRequest;
}

export namespace ListMessagesRequest {
  export type AsObject = {
    pageSize: number,
    pageToken: string,
    epEui: string,
    bsEui: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListMessagesResponse extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<Message>;
  setMessagesList(value: Array<Message>): void;
  addMessages(value?: Message, index?: number): Message;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListMessagesResponse): ListMessagesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListMessagesResponse;
  static deserializeBinaryFromReader(message: ListMessagesResponse, reader: jspb.BinaryReader): ListMessagesResponse;
}

export namespace ListMessagesResponse {
  export type AsObject = {
    messagesList: Array<Message.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class ListEndpointMessagesRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndpointMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndpointMessagesRequest): ListEndpointMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndpointMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndpointMessagesRequest;
  static deserializeBinaryFromReader(message: ListEndpointMessagesRequest, reader: jspb.BinaryReader): ListEndpointMessagesRequest;
}

export namespace ListEndpointMessagesRequest {
  export type AsObject = {
    epEui: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListEndpointMessagesResponse extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<Message>;
  setMessagesList(value: Array<Message>): void;
  addMessages(value?: Message, index?: number): Message;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEndpointMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListEndpointMessagesResponse): ListEndpointMessagesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListEndpointMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEndpointMessagesResponse;
  static deserializeBinaryFromReader(message: ListEndpointMessagesResponse, reader: jspb.BinaryReader): ListEndpointMessagesResponse;
}

export namespace ListEndpointMessagesResponse {
  export type AsObject = {
    messagesList: Array<Message.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class StreamMessagesRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreamMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StreamMessagesRequest): StreamMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StreamMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreamMessagesRequest;
  static deserializeBinaryFromReader(message: StreamMessagesRequest, reader: jspb.BinaryReader): StreamMessagesRequest;
}

export namespace StreamMessagesRequest {
  export type AsObject = {
    epEui: string,
    bsEui: string,
  }
}

export class ListBaseStationMessagesRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDirection(): string;
  setDirection(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationMessagesRequest): ListBaseStationMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationMessagesRequest;
  static deserializeBinaryFromReader(message: ListBaseStationMessagesRequest, reader: jspb.BinaryReader): ListBaseStationMessagesRequest;
}

export namespace ListBaseStationMessagesRequest {
  export type AsObject = {
    bsEui: string,
    pageSize: number,
    pageToken: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    direction: string,
    epEui: string,
  }
}

export class ListBaseStationMessagesResponse extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<BaseStationMessage>;
  setMessagesList(value: Array<BaseStationMessage>): void;
  addMessages(value?: BaseStationMessage, index?: number): BaseStationMessage;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBaseStationMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBaseStationMessagesResponse): ListBaseStationMessagesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListBaseStationMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBaseStationMessagesResponse;
  static deserializeBinaryFromReader(message: ListBaseStationMessagesResponse, reader: jspb.BinaryReader): ListBaseStationMessagesResponse;
}

export namespace ListBaseStationMessagesResponse {
  export type AsObject = {
    messagesList: Array<BaseStationMessage.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class GetBaseStationMessageRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getMessageId(): string;
  setMessageId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessageRequest): GetBaseStationMessageRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessageRequest;
  static deserializeBinaryFromReader(message: GetBaseStationMessageRequest, reader: jspb.BinaryReader): GetBaseStationMessageRequest;
}

export namespace GetBaseStationMessageRequest {
  export type AsObject = {
    bsEui: string,
    messageId: string,
  }
}

export class GetBaseStationMessageResponse extends jspb.Message {
  hasMessage(): boolean;
  clearMessage(): void;
  getMessage(): BaseStationMessage | undefined;
  setMessage(value?: BaseStationMessage): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessageResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessageResponse): GetBaseStationMessageResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessageResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessageResponse;
  static deserializeBinaryFromReader(message: GetBaseStationMessageResponse, reader: jspb.BinaryReader): GetBaseStationMessageResponse;
}

export namespace GetBaseStationMessageResponse {
  export type AsObject = {
    message?: BaseStationMessage.AsObject,
  }
}

export class GetBaseStationMessageStatsRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessageStatsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessageStatsRequest): GetBaseStationMessageStatsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessageStatsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessageStatsRequest;
  static deserializeBinaryFromReader(message: GetBaseStationMessageStatsRequest, reader: jspb.BinaryReader): GetBaseStationMessageStatsRequest;
}

export namespace GetBaseStationMessageStatsRequest {
  export type AsObject = {
    bsEui: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetBaseStationMessageStatsResponse extends jspb.Message {
  hasStats(): boolean;
  clearStats(): void;
  getStats(): BaseStationMessageStats | undefined;
  setStats(value?: BaseStationMessageStats): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBaseStationMessageStatsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetBaseStationMessageStatsResponse): GetBaseStationMessageStatsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetBaseStationMessageStatsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBaseStationMessageStatsResponse;
  static deserializeBinaryFromReader(message: GetBaseStationMessageStatsResponse, reader: jspb.BinaryReader): GetBaseStationMessageStatsResponse;
}

export namespace GetBaseStationMessageStatsResponse {
  export type AsObject = {
    stats?: BaseStationMessageStats.AsObject,
  }
}

export class SearchBaseStationMessagesRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getQuery(): string;
  setQuery(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDirection(): string;
  setDirection(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SearchBaseStationMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SearchBaseStationMessagesRequest): SearchBaseStationMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SearchBaseStationMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SearchBaseStationMessagesRequest;
  static deserializeBinaryFromReader(message: SearchBaseStationMessagesRequest, reader: jspb.BinaryReader): SearchBaseStationMessagesRequest;
}

export namespace SearchBaseStationMessagesRequest {
  export type AsObject = {
    bsEui: string,
    query: string,
    pageSize: number,
    pageToken: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    direction: string,
    epEui: string,
  }
}

export class SearchBaseStationMessagesResponse extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<BaseStationMessage>;
  setMessagesList(value: Array<BaseStationMessage>): void;
  addMessages(value?: BaseStationMessage, index?: number): BaseStationMessage;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SearchBaseStationMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SearchBaseStationMessagesResponse): SearchBaseStationMessagesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SearchBaseStationMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SearchBaseStationMessagesResponse;
  static deserializeBinaryFromReader(message: SearchBaseStationMessagesResponse, reader: jspb.BinaryReader): SearchBaseStationMessagesResponse;
}

export namespace SearchBaseStationMessagesResponse {
  export type AsObject = {
    messagesList: Array<BaseStationMessage.AsObject>,
    nextPageToken: string,
    totalCount: number,
  }
}

export class ExportBaseStationMessagesRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getFormat(): string;
  setFormat(value: string): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasEndTime(): boolean;
  clearEndTime(): void;
  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDirection(): string;
  setDirection(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExportBaseStationMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ExportBaseStationMessagesRequest): ExportBaseStationMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExportBaseStationMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExportBaseStationMessagesRequest;
  static deserializeBinaryFromReader(message: ExportBaseStationMessagesRequest, reader: jspb.BinaryReader): ExportBaseStationMessagesRequest;
}

export namespace ExportBaseStationMessagesRequest {
  export type AsObject = {
    bsEui: string,
    format: string,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    direction: string,
    epEui: string,
  }
}

export class ExportBaseStationMessagesResponse extends jspb.Message {
  getContent(): Uint8Array | string;
  getContent_asU8(): Uint8Array;
  getContent_asB64(): string;
  setContent(value: Uint8Array | string): void;

  getFilename(): string;
  setFilename(value: string): void;

  getContentType(): string;
  setContentType(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExportBaseStationMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ExportBaseStationMessagesResponse): ExportBaseStationMessagesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExportBaseStationMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExportBaseStationMessagesResponse;
  static deserializeBinaryFromReader(message: ExportBaseStationMessagesResponse, reader: jspb.BinaryReader): ExportBaseStationMessagesResponse;
}

export namespace ExportBaseStationMessagesResponse {
  export type AsObject = {
    content: Uint8Array | string,
    filename: string,
    contentType: string,
  }
}

export class StreamBaseStationMessagesRequest extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getDirection(): string;
  setDirection(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreamBaseStationMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StreamBaseStationMessagesRequest): StreamBaseStationMessagesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StreamBaseStationMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreamBaseStationMessagesRequest;
  static deserializeBinaryFromReader(message: StreamBaseStationMessagesRequest, reader: jspb.BinaryReader): StreamBaseStationMessagesRequest;
}

export namespace StreamBaseStationMessagesRequest {
  export type AsObject = {
    bsEui: string,
    direction: string,
    epEui: string,
  }
}

export class BaseStationMessage extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getBsEui(): string;
  setBsEui(value: string): void;

  getEpEui(): string;
  setEpEui(value: string): void;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): void;

  getRssi(): number;
  setRssi(value: number): void;

  getSnr(): number;
  setSnr(value: number): void;

  getEqSnr(): number;
  setEqSnr(value: number): void;

  getUplinkMode(): string;
  setUplinkMode(value: string): void;

  getPacketCounter(): number;
  setPacketCounter(value: number): void;

  hasReceivedAt(): boolean;
  clearReceivedAt(): void;
  getReceivedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setReceivedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getDirection(): string;
  setDirection(value: string): void;

  clearBaseStationsList(): void;
  getBaseStationsList(): Array<BaseStationReceptionInfo>;
  setBaseStationsList(value: Array<BaseStationReceptionInfo>): void;
  addBaseStations(value?: BaseStationReceptionInfo, index?: number): BaseStationReceptionInfo;

  getDuplicate(): boolean;
  setDuplicate(value: boolean): void;

  getDecodedPayload(): Uint8Array | string;
  getDecodedPayload_asU8(): Uint8Array;
  getDecodedPayload_asB64(): string;
  setDecodedPayload(value: Uint8Array | string): void;

  getDecodeStatus(): string;
  setDecodeStatus(value: string): void;

  getDecodeErrorCode(): string;
  setDecodeErrorCode(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationMessage.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationMessage): BaseStationMessage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationMessage;
  static deserializeBinaryFromReader(message: BaseStationMessage, reader: jspb.BinaryReader): BaseStationMessage;
}

export namespace BaseStationMessage {
  export type AsObject = {
    id: string,
    bsEui: string,
    epEui: string,
    payload: Uint8Array | string,
    rssi: number,
    snr: number,
    eqSnr: number,
    uplinkMode: string,
    packetCounter: number,
    receivedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    direction: string,
    baseStationsList: Array<BaseStationReceptionInfo.AsObject>,
    duplicate: boolean,
    decodedPayload: Uint8Array | string,
    decodeStatus: string,
    decodeErrorCode: string,
  }
}

export class BaseStationMessageStats extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getTotalMessages(): number;
  setTotalMessages(value: number): void;

  getUniqueEndpoints(): number;
  setUniqueEndpoints(value: number): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  getMessagesToday(): number;
  setMessagesToday(value: number): void;

  getMessagesThisWeek(): number;
  setMessagesThisWeek(value: number): void;

  getMessagesThisMonth(): number;
  setMessagesThisMonth(value: number): void;

  hasFirstMessageAt(): boolean;
  clearFirstMessageAt(): void;
  getFirstMessageAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstMessageAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastMessageAt(): boolean;
  clearLastMessageAt(): void;
  getLastMessageAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastMessageAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationMessageStats.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationMessageStats): BaseStationMessageStats.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationMessageStats, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationMessageStats;
  static deserializeBinaryFromReader(message: BaseStationMessageStats, reader: jspb.BinaryReader): BaseStationMessageStats;
}

export namespace BaseStationMessageStats {
  export type AsObject = {
    bsEui: string,
    totalMessages: number,
    uniqueEndpoints: number,
    avgRssi: number,
    avgSnr: number,
    messagesToday: number,
    messagesThisWeek: number,
    messagesThisMonth: number,
    firstMessageAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastMessageAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class GetEndPointStatsRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEndPointStatsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEndPointStatsRequest): GetEndPointStatsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEndPointStatsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEndPointStatsRequest;
  static deserializeBinaryFromReader(message: GetEndPointStatsRequest, reader: jspb.BinaryReader): GetEndPointStatsRequest;
}

export namespace GetEndPointStatsRequest {
  export type AsObject = {
    epEui: string,
  }
}

export class GetEndPointStatsResponse extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  getUniqueEndpoints(): number;
  setUniqueEndpoints(value: number): void;

  getAvgRssi(): number;
  setAvgRssi(value: number): void;

  getAvgSnr(): number;
  setAvgSnr(value: number): void;

  hasFirstSeen(): boolean;
  clearFirstSeen(): void;
  getFirstSeen(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstSeen(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastSeen(): boolean;
  clearLastSeen(): void;
  getLastSeen(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastSeen(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getActiveDays(): number;
  setActiveDays(value: number): void;

  getAttachStatus(): string;
  setAttachStatus(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEndPointStatsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetEndPointStatsResponse): GetEndPointStatsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEndPointStatsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEndPointStatsResponse;
  static deserializeBinaryFromReader(message: GetEndPointStatsResponse, reader: jspb.BinaryReader): GetEndPointStatsResponse;
}

export namespace GetEndPointStatsResponse {
  export type AsObject = {
    epEui: string,
    totalCount: number,
    uniqueEndpoints: number,
    avgRssi: number,
    avgSnr: number,
    firstSeen?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastSeen?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    activeDays: number,
    attachStatus: string,
  }
}

export class GetEndPointOperationsRequest extends jspb.Message {
  getEpEui(): string;
  setEpEui(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEndPointOperationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEndPointOperationsRequest): GetEndPointOperationsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEndPointOperationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEndPointOperationsRequest;
  static deserializeBinaryFromReader(message: GetEndPointOperationsRequest, reader: jspb.BinaryReader): GetEndPointOperationsRequest;
}

export namespace GetEndPointOperationsRequest {
  export type AsObject = {
    epEui: string,
    pageSize: number,
    offset: number,
  }
}

export class GetEndPointOperationsResponse extends jspb.Message {
  clearOperationsList(): void;
  getOperationsList(): Array<EndPointOperation>;
  setOperationsList(value: Array<EndPointOperation>): void;
  addOperations(value?: EndPointOperation, index?: number): EndPointOperation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEndPointOperationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetEndPointOperationsResponse): GetEndPointOperationsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEndPointOperationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEndPointOperationsResponse;
  static deserializeBinaryFromReader(message: GetEndPointOperationsResponse, reader: jspb.BinaryReader): GetEndPointOperationsResponse;
}

export namespace GetEndPointOperationsResponse {
  export type AsObject = {
    operationsList: Array<EndPointOperation.AsObject>,
  }
}

export class EndPointOperation extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEventType(): string;
  setEventType(value: string): void;

  getCategory(): string;
  setCategory(value: string): void;

  getSeverity(): string;
  setSeverity(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EndPointOperation.AsObject;
  static toObject(includeInstance: boolean, msg: EndPointOperation): EndPointOperation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EndPointOperation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EndPointOperation;
  static deserializeBinaryFromReader(message: EndPointOperation, reader: jspb.BinaryReader): EndPointOperation;
}

export namespace EndPointOperation {
  export type AsObject = {
    id: string,
    eventType: string,
    category: string,
    severity: string,
    title: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListAllBaseStationLocationsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllBaseStationLocationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllBaseStationLocationsRequest): ListAllBaseStationLocationsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListAllBaseStationLocationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllBaseStationLocationsRequest;
  static deserializeBinaryFromReader(message: ListAllBaseStationLocationsRequest, reader: jspb.BinaryReader): ListAllBaseStationLocationsRequest;
}

export namespace ListAllBaseStationLocationsRequest {
  export type AsObject = {
  }
}

export class BaseStationLocation extends jspb.Message {
  getBsEui(): string;
  setBsEui(value: string): void;

  getName(): string;
  setName(value: string): void;

  getLatitude(): number;
  setLatitude(value: number): void;

  getLongitude(): number;
  setLongitude(value: number): void;

  hasAltitude(): boolean;
  clearAltitude(): void;
  getAltitude(): google_protobuf_wrappers_pb.DoubleValue | undefined;
  setAltitude(value?: google_protobuf_wrappers_pb.DoubleValue): void;

  getLocationSource(): string;
  setLocationSource(value: string): void;

  getIsOnline(): boolean;
  setIsOnline(value: boolean): void;

  getOrgId(): string;
  setOrgId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BaseStationLocation.AsObject;
  static toObject(includeInstance: boolean, msg: BaseStationLocation): BaseStationLocation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BaseStationLocation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BaseStationLocation;
  static deserializeBinaryFromReader(message: BaseStationLocation, reader: jspb.BinaryReader): BaseStationLocation;
}

export namespace BaseStationLocation {
  export type AsObject = {
    bsEui: string,
    name: string,
    latitude: number,
    longitude: number,
    altitude?: google_protobuf_wrappers_pb.DoubleValue.AsObject,
    locationSource: string,
    isOnline: boolean,
    orgId: string,
  }
}

export class ListAllBaseStationLocationsResponse extends jspb.Message {
  clearLocationsList(): void;
  getLocationsList(): Array<BaseStationLocation>;
  setLocationsList(value: Array<BaseStationLocation>): void;
  addLocations(value?: BaseStationLocation, index?: number): BaseStationLocation;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllBaseStationLocationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllBaseStationLocationsResponse): ListAllBaseStationLocationsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListAllBaseStationLocationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllBaseStationLocationsResponse;
  static deserializeBinaryFromReader(message: ListAllBaseStationLocationsResponse, reader: jspb.BinaryReader): ListAllBaseStationLocationsResponse;
}

export namespace ListAllBaseStationLocationsResponse {
  export type AsObject = {
    locationsList: Array<BaseStationLocation.AsObject>,
    totalCount: number,
  }
}

export class GetCEStatusRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCEStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetCEStatusRequest): GetCEStatusRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetCEStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCEStatusRequest;
  static deserializeBinaryFromReader(message: GetCEStatusRequest, reader: jspb.BinaryReader): GetCEStatusRequest;
}

export namespace GetCEStatusRequest {
  export type AsObject = {
  }
}

export class GetCEStatusResponse extends jspb.Message {
  getOnboardingRequired(): boolean;
  setOnboardingRequired(value: boolean): void;

  getCeId(): string;
  setCeId(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  getFederationConnected(): boolean;
  setFederationConnected(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCEStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetCEStatusResponse): GetCEStatusResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetCEStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCEStatusResponse;
  static deserializeBinaryFromReader(message: GetCEStatusResponse, reader: jspb.BinaryReader): GetCEStatusResponse;
}

export namespace GetCEStatusResponse {
  export type AsObject = {
    onboardingRequired: boolean,
    ceId: string,
    companyName: string,
    federationConnected: boolean,
  }
}

export class CompleteCEOnboardingRequest extends jspb.Message {
  getCompanyName(): string;
  setCompanyName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompleteCEOnboardingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CompleteCEOnboardingRequest): CompleteCEOnboardingRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CompleteCEOnboardingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompleteCEOnboardingRequest;
  static deserializeBinaryFromReader(message: CompleteCEOnboardingRequest, reader: jspb.BinaryReader): CompleteCEOnboardingRequest;
}

export namespace CompleteCEOnboardingRequest {
  export type AsObject = {
    companyName: string,
  }
}

export class CompleteCEOnboardingResponse extends jspb.Message {
  getCeId(): string;
  setCeId(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompleteCEOnboardingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CompleteCEOnboardingResponse): CompleteCEOnboardingResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CompleteCEOnboardingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompleteCEOnboardingResponse;
  static deserializeBinaryFromReader(message: CompleteCEOnboardingResponse, reader: jspb.BinaryReader): CompleteCEOnboardingResponse;
}

export namespace CompleteCEOnboardingResponse {
  export type AsObject = {
    ceId: string,
    companyName: string,
  }
}

export class CEInstanceInfo extends jspb.Message {
  getCeId(): string;
  setCeId(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getConnectionStatus(): string;
  setConnectionStatus(value: string): void;

  getActiveBasestationCount(): number;
  setActiveBasestationCount(value: number): void;

  getTotalPacketsRelayed(): number;
  setTotalPacketsRelayed(value: number): void;

  getCeVersion(): string;
  setCeVersion(value: string): void;

  hasFirstSeenAt(): boolean;
  clearFirstSeenAt(): void;
  getFirstSeenAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstSeenAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastHeartbeatAt(): boolean;
  clearLastHeartbeatAt(): void;
  getLastHeartbeatAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastHeartbeatAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CEInstanceInfo.AsObject;
  static toObject(includeInstance: boolean, msg: CEInstanceInfo): CEInstanceInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CEInstanceInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CEInstanceInfo;
  static deserializeBinaryFromReader(message: CEInstanceInfo, reader: jspb.BinaryReader): CEInstanceInfo;
}

export namespace CEInstanceInfo {
  export type AsObject = {
    ceId: string,
    companyName: string,
    status: string,
    connectionStatus: string,
    activeBasestationCount: number,
    totalPacketsRelayed: number,
    ceVersion: string,
    firstSeenAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastHeartbeatAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ListCEInstancesRequest extends jspb.Message {
  getStatusFilter(): string;
  setStatusFilter(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCEInstancesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListCEInstancesRequest): ListCEInstancesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListCEInstancesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCEInstancesRequest;
  static deserializeBinaryFromReader(message: ListCEInstancesRequest, reader: jspb.BinaryReader): ListCEInstancesRequest;
}

export namespace ListCEInstancesRequest {
  export type AsObject = {
    statusFilter: string,
    pageSize: number,
    pageToken: string,
  }
}

export class ListCEInstancesResponse extends jspb.Message {
  clearInstancesList(): void;
  getInstancesList(): Array<CEInstanceInfo>;
  setInstancesList(value: Array<CEInstanceInfo>): void;
  addInstances(value?: CEInstanceInfo, index?: number): CEInstanceInfo;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCEInstancesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListCEInstancesResponse): ListCEInstancesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ListCEInstancesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCEInstancesResponse;
  static deserializeBinaryFromReader(message: ListCEInstancesResponse, reader: jspb.BinaryReader): ListCEInstancesResponse;
}

export namespace ListCEInstancesResponse {
  export type AsObject = {
    instancesList: Array<CEInstanceInfo.AsObject>,
    totalCount: number,
    nextPageToken: string,
  }
}

export class RevokeCEInstanceRequest extends jspb.Message {
  getCeId(): string;
  setCeId(value: string): void;

  getReason(): string;
  setReason(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RevokeCEInstanceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RevokeCEInstanceRequest): RevokeCEInstanceRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RevokeCEInstanceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RevokeCEInstanceRequest;
  static deserializeBinaryFromReader(message: RevokeCEInstanceRequest, reader: jspb.BinaryReader): RevokeCEInstanceRequest;
}

export namespace RevokeCEInstanceRequest {
  export type AsObject = {
    ceId: string,
    reason: string,
  }
}

export class RevokeCEInstanceResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RevokeCEInstanceResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RevokeCEInstanceResponse): RevokeCEInstanceResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RevokeCEInstanceResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RevokeCEInstanceResponse;
  static deserializeBinaryFromReader(message: RevokeCEInstanceResponse, reader: jspb.BinaryReader): RevokeCEInstanceResponse;
}

export namespace RevokeCEInstanceResponse {
  export type AsObject = {
    success: boolean,
  }
}

