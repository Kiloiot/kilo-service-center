/**
 * Device catalog RPCs (CoreService): manufacturers, device models and blueprints.
 */

import * as corePb from "@services/grpc/core_pb";
import { decodeExactJsonObject } from "@utils/exact-json";
import { HTTP_STATUS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { bytesToString, stringToBytes } from "../codec";
import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/**
 * Blueprint transport DTO — maps protobuf fields to a plain transport shape.
 * The RPC layer knows only proto shapes; the api layer maps to UI types.
 */
export interface BlueprintTransportDTO {
  id: string;
  deviceModelId: string;
  name: string;
  version: string;
  description?: string;
  decoderScript: string;
  typeEui: string;
  specJson: string;
  isDefault: boolean;
  isSystem: boolean;
  registryRepo?: string;
  registryCommitSha?: string;
  registryVerified: boolean;
  registryPrUrl?: string;
  createdAt?: Date;
  updatedAt?: Date;
}

/**
 * Maps a protobuf Blueprint message to the transport DTO.
 * Canonical spec_json (field 11) takes precedence; falls back to decoder_script (field 6).
 */
export function mapProtoBlueprintToTransport(
  b: corePb.Blueprint,
): BlueprintTransportDTO {
  const specJsonRaw = bytesToString(b.getSpecJson_asU8());
  const decoderScriptRaw = bytesToString(b.getDecoderScript_asU8());
  const specJson = specJsonRaw || decoderScriptRaw;

  return {
    id: b.getId(),
    deviceModelId: b.getDeviceModelId(),
    name: b.getName(),
    version: b.getVersion(),
    description: b.getDescription() || undefined,
    decoderScript: decoderScriptRaw,
    typeEui: b.getTypeEui(),
    specJson,
    isDefault: b.getIsDefault(),
    isSystem: b.getIsSystem(),
    registryRepo: b.getRegistryRepo() || undefined,
    registryCommitSha: b.getRegistryCommitSha() || undefined,
    registryVerified: b.getRegistryVerified(),
    registryPrUrl: b.getRegistryPrUrl() || undefined,
    createdAt: b.getCreatedAt()?.toDate(),
    updatedAt: b.getUpdatedAt()?.toDate(),
  };
}

/**
 * List manufacturers
 */
export async function listManufacturers(isSystem = false): Promise<
  Array<{
    id: string;
    name: string;
    code: string;
    description?: string;
    website?: string;
    contactEmail?: string;
    tenantId: string;
    isVerified: boolean;
    isSystem: boolean;
    modelCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  }>
> {
  const request = new corePb.ListManufacturersRequest();
  request.setIsSystem(isSystem);

  const response = await grpcTransport.callCore<
    corePb.ListManufacturersRequest,
    corePb.ListManufacturersResponse
  >((c) => c.listManufacturers, request);

  return response.getManufacturersList().map((m) => ({
    id: m.getId(),
    name: m.getName(),
    code: m.getCode(),
    description: m.getDescription() || undefined,
    website: m.getWebsite() || undefined,
    contactEmail: m.getContactEmail() || undefined,
    tenantId: m.getTenantId(),
    isVerified: m.getIsVerified(),
    isSystem: m.getIsSystem(),
    modelCount: m.getModelCount(),
    createdAt: m.getCreatedAt()?.toDate(),
    updatedAt: m.getUpdatedAt()?.toDate(),
  }));
}

/**
 * Get manufacturer by ID
 */
export async function getManufacturer(manufacturerId: string): Promise<{
  id: string;
  name: string;
  code: string;
  description?: string;
  website?: string;
  contactEmail?: string;
  tenantId: string;
  isVerified: boolean;
  isSystem: boolean;
  modelCount: number;
  createdAt?: Date;
  updatedAt?: Date;
} | null> {
  const request = new corePb.GetManufacturerRequest();
  request.setId(manufacturerId);

  try {
    const response = await grpcTransport.callCore<
      corePb.GetManufacturerRequest,
      corePb.GetManufacturerResponse
    >((c) => c.getManufacturer, request);

    const m = response.getManufacturer();
    if (!m) return null;

    return {
      id: m.getId(),
      name: m.getName(),
      code: m.getCode(),
      description: m.getDescription() || undefined,
      website: m.getWebsite() || undefined,
      contactEmail: m.getContactEmail() || undefined,
      tenantId: m.getTenantId(),
      isVerified: m.getIsVerified(),
      isSystem: m.getIsSystem(),
      modelCount: m.getModelCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    };
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Create manufacturer
 */
export async function createManufacturer(data: {
  name: string;
  code?: string;
  description?: string;
  website?: string;
  contactEmail?: string;
  isSystem?: boolean;
}): Promise<{
  id: string;
  name: string;
  code: string;
  description?: string;
  website?: string;
  contactEmail?: string;
  tenantId: string;
  isVerified: boolean;
  isSystem: boolean;
  modelCount: number;
  createdAt?: Date;
  updatedAt?: Date;
}> {
  const request = new corePb.CreateManufacturerRequest();
  request.setName(data.name);
  if (data.code) request.setCode(data.code);
  if (data.description) request.setDescription(data.description);
  if (data.website) request.setWebsite(data.website);
  if (data.contactEmail) request.setContactEmail(data.contactEmail);
  if (data.isSystem) request.setIsSystem(data.isSystem);

  const response = await grpcTransport.callCore<
    corePb.CreateManufacturerRequest,
    corePb.CreateManufacturerResponse
  >((c) => c.createManufacturer, request);

  const m = response.getManufacturer();
  if (!m) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_CREATE_MANUFACTURER_RESPONSE,
    );
  }

  return {
    id: m.getId(),
    name: m.getName(),
    code: m.getCode(),
    description: m.getDescription() || undefined,
    website: m.getWebsite() || undefined,
    contactEmail: m.getContactEmail() || undefined,
    tenantId: m.getTenantId(),
    isVerified: m.getIsVerified(),
    isSystem: m.getIsSystem(),
    modelCount: m.getModelCount(),
    createdAt: m.getCreatedAt()?.toDate(),
    updatedAt: m.getUpdatedAt()?.toDate(),
  };
}

/**
 * Update manufacturer
 */
export async function updateManufacturer(
  manufacturerId: string,
  data: {
    name?: string;
    description?: string;
    website?: string;
    contactEmail?: string;
  },
): Promise<void> {
  const request = new corePb.UpdateManufacturerRequest();
  request.setId(manufacturerId);
  if (data.name) request.setName(data.name);
  if (data.description !== undefined) request.setDescription(data.description);
  if (data.website !== undefined) request.setWebsite(data.website);
  if (data.contactEmail !== undefined)
    request.setContactEmail(data.contactEmail);

  await grpcTransport.callCore<
    corePb.UpdateManufacturerRequest,
    corePb.UpdateManufacturerResponse
  >((c) => c.updateManufacturer, request);
}

/**
 * Delete manufacturer
 */
export async function deleteManufacturer(
  manufacturerId: string,
): Promise<void> {
  const request = new corePb.DeleteManufacturerRequest();
  request.setId(manufacturerId);

  await grpcTransport.callCore<
    corePb.DeleteManufacturerRequest,
    corePb.DeleteManufacturerResponse
  >((c) => c.deleteManufacturer, request);
}

/**
 * List device models for a manufacturer
 */
export async function listDeviceModels(
  manufacturerId: string,
  isSystem = false,
): Promise<
  Array<{
    id: string;
    manufacturerId: string;
    tenantId: string;
    name: string;
    code: string;
    typeEui?: string;
    description?: string;
    datasheetUrl?: string;
    isSystem: boolean;
    blueprintCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  }>
> {
  const request = new corePb.ListDeviceModelsRequest();
  request.setManufacturerId(manufacturerId);
  request.setIsSystem(isSystem);

  const response = await grpcTransport.callCore<
    corePb.ListDeviceModelsRequest,
    corePb.ListDeviceModelsResponse
  >((c) => c.listDeviceModels, request);

  return response.getDeviceModelsList().map((m) => ({
    id: m.getId(),
    manufacturerId: m.getManufacturerId(),
    tenantId: m.getTenantId(),
    name: m.getName(),
    code: m.getCode(),
    typeEui: m.getTypeEui() || undefined,
    description: m.getDescription() || undefined,
    datasheetUrl: m.getDatasheetUrl() || undefined,
    isSystem: m.getIsSystem(),
    blueprintCount: m.getBlueprintCount(),
    createdAt: m.getCreatedAt()?.toDate(),
    updatedAt: m.getUpdatedAt()?.toDate(),
  }));
}

/**
 * Get device model by ID
 */
export async function getDeviceModel(modelId: string): Promise<{
  id: string;
  manufacturerId: string;
  tenantId: string;
  name: string;
  code: string;
  typeEui?: string;
  description?: string;
  datasheetUrl?: string;
  isSystem: boolean;
  blueprintCount: number;
  createdAt?: Date;
  updatedAt?: Date;
} | null> {
  const request = new corePb.GetDeviceModelRequest();
  request.setId(modelId);

  try {
    const response = await grpcTransport.callCore<
      corePb.GetDeviceModelRequest,
      corePb.GetDeviceModelResponse
    >((c) => c.getDeviceModel, request);

    const m = response.getDeviceModel();
    if (!m) return null;

    return {
      id: m.getId(),
      manufacturerId: m.getManufacturerId(),
      tenantId: m.getTenantId(),
      name: m.getName(),
      code: m.getCode(),
      typeEui: m.getTypeEui() || undefined,
      description: m.getDescription() || undefined,
      datasheetUrl: m.getDatasheetUrl() || undefined,
      isSystem: m.getIsSystem(),
      blueprintCount: m.getBlueprintCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    };
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Update device model
 */
export async function updateDeviceModel(
  modelId: string,
  data: {
    name?: string;
    description?: string;
  },
): Promise<void> {
  const request = new corePb.UpdateDeviceModelRequest();
  request.setId(modelId);
  if (data.name) request.setName(data.name);
  if (data.description !== undefined) request.setDescription(data.description);

  await grpcTransport.callCore<
    corePb.UpdateDeviceModelRequest,
    corePb.UpdateDeviceModelResponse
  >((c) => c.updateDeviceModel, request);
}

/**
 * Delete device model
 */
export async function deleteDeviceModel(modelId: string): Promise<void> {
  const request = new corePb.DeleteDeviceModelRequest();
  request.setId(modelId);

  await grpcTransport.callCore<
    corePb.DeleteDeviceModelRequest,
    corePb.DeleteDeviceModelResponse
  >((c) => c.deleteDeviceModel, request);
}

/**
 * Create device model with default blueprint atomically
 */
export async function createDeviceModelWithBlueprint(data: {
  manufacturerId: string;
  name: string;
  version: string;
  decoderScript: string;
  isSystem?: boolean;
}): Promise<{
  deviceModel: {
    id: string;
    manufacturerId: string;
    tenantId: string;
    name: string;
    code: string;
    typeEui: string;
    description: string;
    datasheetUrl: string;
    isSystem: boolean;
    blueprintCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  };
  blueprint: BlueprintTransportDTO;
}> {
  const request = new corePb.CreateDeviceModelWithBlueprintRequest();
  request.setManufacturerId(data.manufacturerId);
  request.setName(data.name);
  request.setVersion(data.version);
  request.setDecoderScript(stringToBytes(data.decoderScript));
  if (data.isSystem) request.setIsSystem(data.isSystem);

  const response = await grpcTransport.callCore<
    corePb.CreateDeviceModelWithBlueprintRequest,
    corePb.CreateDeviceModelWithBlueprintResponse
  >((c) => c.createDeviceModelWithBlueprint, request);

  const model = response.getDeviceModel();
  const blueprint = response.getBlueprint();
  if (!model || !blueprint) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_CREATE_BLUEPRINT_RESPONSE,
    );
  }

  return {
    deviceModel: {
      id: model.getId(),
      manufacturerId: model.getManufacturerId(),
      tenantId: model.getTenantId(),
      name: model.getName(),
      code: model.getCode(),
      typeEui: model.getTypeEui(),
      description: model.getDescription(),
      datasheetUrl: model.getDatasheetUrl(),
      isSystem: model.getIsSystem(),
      blueprintCount: model.getBlueprintCount(),
      createdAt: model.getCreatedAt()?.toDate(),
      updatedAt: model.getUpdatedAt()?.toDate(),
    },
    blueprint: mapProtoBlueprintToTransport(blueprint),
  };
}

/**
 * List blueprints for a device model
 */
export async function listBlueprints(
  modelId: string,
  isSystem = false,
): Promise<Array<BlueprintTransportDTO>> {
  const request = new corePb.ListBlueprintsRequest();
  request.setDeviceModelId(modelId);
  request.setIsSystem(isSystem);

  const response = await grpcTransport.callCore<
    corePb.ListBlueprintsRequest,
    corePb.ListBlueprintsResponse
  >((c) => c.listBlueprints, request);

  return response
    .getBlueprintsList()
    .map((b) => mapProtoBlueprintToTransport(b));
}

/**
 * Get blueprint by ID
 */
export async function getBlueprint(
  blueprintId: string,
): Promise<BlueprintTransportDTO | null> {
  const request = new corePb.GetBlueprintRequest();
  request.setId(blueprintId);

  try {
    const response = await grpcTransport.callCore<
      corePb.GetBlueprintRequest,
      corePb.GetBlueprintResponse
    >((c) => c.getBlueprint, request);

    const b = response.getBlueprint();
    if (!b) return null;

    return mapProtoBlueprintToTransport(b);
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Create blueprint
 */
export async function createBlueprint(
  modelId: string,
  data: {
    name: string;
    version: string;
    description?: string;
    decoderScript: string;
    isDefault?: boolean;
    isSystem?: boolean;
  },
): Promise<BlueprintTransportDTO> {
  const request = new corePb.CreateBlueprintRequest();
  request.setDeviceModelId(modelId);
  request.setName(data.name);
  request.setVersion(data.version);
  request.setDecoderScript(stringToBytes(data.decoderScript));
  if (data.description) request.setDescription(data.description);
  if (data.isDefault !== undefined) request.setIsDefault(data.isDefault);
  if (data.isSystem) request.setIsSystem(data.isSystem);

  const response = await grpcTransport.callCore<
    corePb.CreateBlueprintRequest,
    corePb.CreateBlueprintResponse
  >((c) => c.createBlueprint, request);

  const b = response.getBlueprint();
  if (!b) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_CREATE_BLUEPRINT_RESPONSE,
    );
  }

  return mapProtoBlueprintToTransport(b);
}

/**
 * Update blueprint
 */
export async function updateBlueprint(
  blueprintId: string,
  data: {
    name?: string;
    version?: string;
    description?: string;
    decoderScript?: string;
  },
): Promise<void> {
  const request = new corePb.UpdateBlueprintRequest();
  request.setId(blueprintId);
  if (data.name) request.setName(data.name);
  if (data.version) request.setVersion(data.version);
  if (data.description !== undefined) request.setDescription(data.description);
  if (data.decoderScript)
    request.setDecoderScript(stringToBytes(data.decoderScript));

  await grpcTransport.callCore<
    corePb.UpdateBlueprintRequest,
    corePb.UpdateBlueprintResponse
  >((c) => c.updateBlueprint, request);
}

/**
 * Set blueprint as default
 */
export async function setDefaultBlueprint(blueprintId: string): Promise<void> {
  const request = new corePb.SetDefaultBlueprintRequest();
  request.setId(blueprintId);

  await grpcTransport.callCore<
    corePb.SetDefaultBlueprintRequest,
    corePb.SetDefaultBlueprintResponse
  >((c) => c.setDefaultBlueprint, request);
}

/**
 * Submit blueprint to registry
 */
export async function submitBlueprintToRegistry(
  blueprintId: string,
  data: {
    contributorName?: string;
    contributorEmail?: string;
    notes?: string;
  },
): Promise<{
  success: boolean;
  prUrl: string;
  commitSha: string;
  branchName: string;
}> {
  const request = new corePb.SubmitBlueprintToRegistryRequest();
  request.setId(blueprintId);
  if (data.contributorName) request.setContributorName(data.contributorName);
  if (data.contributorEmail) request.setContributorEmail(data.contributorEmail);
  if (data.notes) request.setNotes(data.notes);

  const response = await grpcTransport.callCore<
    corePb.SubmitBlueprintToRegistryRequest,
    corePb.SubmitBlueprintToRegistryResponse
  >((c) => c.submitBlueprintToRegistry, request);

  return {
    success: response.getSuccess(),
    prUrl: response.getPrUrl(),
    commitSha: response.getCommitSha(),
    branchName: response.getBranchName(),
  };
}

/**
 * Preview blueprint decoding against a raw payload
 */
export async function decodePreview(
  blueprintId: string,
  payload: Uint8Array,
  formatId: number,
): Promise<{
  success: boolean;
  decodedPayload?: Record<string, unknown>;
  errorCode?: string;
  errorDetail?: string;
  formatId: number;
  blueprintVersion?: string;
}> {
  const request = new corePb.DecodePreviewRequest();
  request.setBlueprintId(blueprintId);
  request.setPayload(payload);
  request.setFormatId(formatId);

  const response = await grpcTransport.callCore<
    corePb.DecodePreviewRequest,
    corePb.DecodePreviewResponse
  >((c) => c.decodePreview, request);

  return {
    success: response.getSuccess(),
    decodedPayload: decodeExactJsonObject(response.getDecodedPayload_asU8()),
    errorCode: response.getErrorCode() || undefined,
    errorDetail: response.getErrorDetail() || undefined,
    formatId: response.getFormatId(),
    blueprintVersion: response.getBlueprintVersion() || undefined,
  };
}

/**
 * Bulk re-materialize snapshot-bearing endpoints of a device model onto a
 * target blueprint. setAsDefault also moves the model's default pointer.
 */
export async function bulkAssignBlueprint(data: {
  blueprintId: string;
  deviceModelId: string;
  epEuis?: string[];
  setAsDefault: boolean;
}): Promise<{ affectedCount: number }> {
  const request = new corePb.BulkAssignBlueprintRequest();
  request.setBlueprintId(data.blueprintId);
  request.setDeviceModelId(data.deviceModelId);
  if (data.epEuis?.length) request.setEpEuisList(data.epEuis);
  request.setSetAsDefault(data.setAsDefault);

  const response = await grpcTransport.callCore<
    corePb.BulkAssignBlueprintRequest,
    corePb.BulkAssignBlueprintResponse
  >((c) => c.bulkAssignBlueprint, request);

  return { affectedCount: response.getAffectedCount() };
}

/**
 * Count snapshot-bearing endpoints for a device model (bulk-migration preview).
 * The device_model_id filter selects only endpoints carrying a snapshot.
 */
export async function countEndpointsByDeviceModel(
  deviceModelId: string,
): Promise<number> {
  const request = new corePb.ListEndPointsRequest();
  request.setDeviceModelId(deviceModelId);

  const response = await grpcTransport.callCore<
    corePb.ListEndPointsRequest,
    corePb.ListEndPointsResponse
  >((c) => c.listEndPoints, request);

  const total = response.getTotalCount();
  return total || response.getEndpointsList().length;
}
