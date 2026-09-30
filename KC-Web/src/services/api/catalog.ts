/**
 * Device catalog (manufacturers, device models, blueprints) in the shapes the UI consumes.
 */

import type {
  BlueprintListItemAPI,
  BlueprintScope,
  BlueprintUI,
  BulkAssignBlueprintResponse,
  CreateBlueprintRequest,
  CreateDeviceModelWithBlueprintRequest,
  CreateManufacturerRequest,
  DecodePreviewRequest,
  DecodePreviewResponse,
  DeviceModelUI,
  ManufacturerUI,
  RegistrySubmitRequest,
  RegistrySubmitResponse,
  UpdateBlueprintRequest,
  UpdateDeviceModelRequest,
  UpdateManufacturerRequest,
} from "@api-types/api";

import type { BlueprintTransportDTO } from "@services/grpc/rpc/catalog";
import * as catalogRpc from "@services/grpc/rpc/catalog";
import { hexToBytes } from "@utils/formatters";
import { BLUEPRINT_SCOPE, ENCODING, REGISTRY_DEFAULTS } from "@constants/app";
import { VAL_INVALID_HEX_STRING } from "@constants/messages";

/** Parse proto tenant IDs (string) safely for UI models. */
export function parseTenantIdOrZero(tenantId: string): number {
  const parsed = Number.parseInt(tenantId, 10);
  return Number.isNaN(parsed) ? 0 : parsed;
}

/** Maps a BlueprintTransportDTO from the gRPC layer to the UI-facing BlueprintUI type */
export function mapTransportBlueprintToUI(
  b: BlueprintTransportDTO,
): BlueprintUI {
  let specJson: Record<string, unknown> = {};
  if (b.specJson) {
    try {
      specJson = JSON.parse(b.specJson);
    } catch {
      specJson = { script: b.specJson };
    }
  }

  return {
    id: b.id,
    deviceModelId: b.deviceModelId,
    tenantId: 0,
    version: b.version,
    typeEui: b.typeEui,
    specJson,
    isDefault: b.isDefault,
    isSystem: b.isSystem,
    registryRepo: b.registryRepo,
    registryCommitSha: b.registryCommitSha,
    registryVerified: b.registryVerified,
    registryPrUrl: b.registryPrUrl,
    createdAt: b.createdAt?.toISOString() || "",
    updatedAt: b.updatedAt?.toISOString() || "",
  };
}

export const payloadWhitespaceRegex = /\s+/g;

export const payloadHexPrefixRegex = /^0x/i;

export const payloadBase64Regex = /^[A-Za-z0-9+/]+={0,2}$/;

export function normalizePayloadInput(payload: string): string {
  return payload
    .replace(payloadWhitespaceRegex, "")
    .replace(payloadHexPrefixRegex, "");
}

export function base64ToBytes(base64Payload: string): Uint8Array {
  const decoded = atob(base64Payload);
  return Uint8Array.from(decoded, (char) => char.charCodeAt(0));
}

/**
 * Decode preview payload parser:
 * 1) Accepts canonical hex (with optional spaces / 0x prefix)
 * 2) Falls back to base64 for copied payloads from endpoint views
 */
export function parseDecodePreviewPayload(payload: string): Uint8Array {
  const normalized = normalizePayloadInput(payload);

  try {
    return hexToBytes(normalized);
  } catch {
    const looksBase64 =
      normalized.length > 0 &&
      normalized.length % ENCODING.BASE64_BLOCK_LENGTH === 0 &&
      payloadBase64Regex.test(normalized);
    if (looksBase64) {
      return base64ToBytes(normalized);
    }
    throw new Error(VAL_INVALID_HEX_STRING);
  }
}

export const catalogApi = {
  async getManufacturers(scope?: BlueprintScope): Promise<ManufacturerUI[]> {
    const response = await catalogRpc.listManufacturers(
      scope === BLUEPRINT_SCOPE.SYSTEM,
    );
    return response.map((m) => ({
      id: m.id,
      tenantId: parseTenantIdOrZero(m.tenantId),
      name: m.name,
      website: m.website,
      isVerified: m.isVerified,
      isSystem: m.isSystem,
      modelCount: m.modelCount,
      createdAt: m.createdAt?.toISOString() || "",
      updatedAt: m.updatedAt?.toISOString() || "",
    }));
  },

  async getManufacturer(id: string): Promise<ManufacturerUI | null> {
    const response = await catalogRpc.getManufacturer(id);
    if (!response) return null;

    return {
      id: response.id,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      website: response.website,
      isVerified: response.isVerified,
      isSystem: response.isSystem,
      modelCount: response.modelCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  },

  async createManufacturer(
    data: CreateManufacturerRequest,
  ): Promise<ManufacturerUI> {
    const response = await catalogRpc.createManufacturer({
      name: data.name,
      website: data.website,
      isSystem: data.isSystem,
    });

    return {
      id: response.id,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      website: response.website,
      isVerified: response.isVerified,
      isSystem: response.isSystem,
      modelCount: response.modelCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  },

  async updateManufacturer(
    id: string,
    data: UpdateManufacturerRequest,
  ): Promise<void> {
    await catalogRpc.updateManufacturer(id, {
      name: data.name,
      website: data.website,
    });
  },

  async deleteManufacturer(id: string): Promise<void> {
    await catalogRpc.deleteManufacturer(id);
  },

  async getDeviceModels(
    manufacturerId: string,
    scope?: BlueprintScope,
  ): Promise<DeviceModelUI[]> {
    const response = await catalogRpc.listDeviceModels(
      manufacturerId,
      scope === BLUEPRINT_SCOPE.SYSTEM,
    );
    return response.map((m) => ({
      id: m.id,
      manufacturerId: m.manufacturerId,
      tenantId: parseTenantIdOrZero(m.tenantId),
      name: m.name,
      code: m.code,
      typeEui: m.typeEui,
      description: m.description,
      datasheetUrl: m.datasheetUrl,
      isSystem: m.isSystem,
      blueprintCount: m.blueprintCount,
      createdAt: m.createdAt?.toISOString() || "",
      updatedAt: m.updatedAt?.toISOString() || "",
    }));
  },

  async getDeviceModel(id: string): Promise<DeviceModelUI | null> {
    const response = await catalogRpc.getDeviceModel(id);
    if (!response) return null;

    return {
      id: response.id,
      manufacturerId: response.manufacturerId,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      code: response.code,
      typeEui: response.typeEui,
      description: response.description,
      datasheetUrl: response.datasheetUrl,
      isSystem: response.isSystem,
      blueprintCount: response.blueprintCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  },

  async updateDeviceModel(
    id: string,
    data: UpdateDeviceModelRequest,
  ): Promise<void> {
    await catalogRpc.updateDeviceModel(id, {
      name: data.name,
      description: data.description,
    });
  },

  async deleteDeviceModel(id: string): Promise<void> {
    await catalogRpc.deleteDeviceModel(id);
  },

  /**
   * Create a device model with a default blueprint atomically.
   * The server generates the model code slug automatically.
   */
  async createDeviceModelWithBlueprint(
    data: CreateDeviceModelWithBlueprintRequest,
  ): Promise<{
    model: DeviceModelUI;
    blueprint: BlueprintUI;
  }> {
    const response = await catalogRpc.createDeviceModelWithBlueprint({
      manufacturerId: data.manufacturerId,
      name: data.name,
      version: data.version,
      decoderScript: JSON.stringify(data.specJson),
      isSystem: data.isSystem,
    });

    return {
      model: {
        id: response.deviceModel.id,
        manufacturerId: response.deviceModel.manufacturerId,
        tenantId: parseTenantIdOrZero(response.deviceModel.tenantId),
        name: response.deviceModel.name,
        code: response.deviceModel.code,
        typeEui: response.deviceModel.typeEui,
        description: response.deviceModel.description,
        datasheetUrl: response.deviceModel.datasheetUrl,
        isSystem: response.deviceModel.isSystem,
        blueprintCount: response.deviceModel.blueprintCount || 1,
        createdAt: response.deviceModel.createdAt?.toISOString() || "",
        updatedAt: response.deviceModel.updatedAt?.toISOString() || "",
      },
      blueprint: mapTransportBlueprintToUI(response.blueprint),
    };
  },

  async getBlueprints(
    modelId: string,
    scope?: BlueprintScope,
  ): Promise<BlueprintListItemAPI[]> {
    const response = await catalogRpc.listBlueprints(
      modelId,
      scope === BLUEPRINT_SCOPE.SYSTEM,
    );
    return response.map((b: BlueprintTransportDTO) => ({
      id: b.id,
      deviceModelId: b.deviceModelId,
      version: b.version,
      typeEui: b.typeEui,
      isDefault: b.isDefault,
      isSystem: b.isSystem,
      createdAt: b.createdAt?.toISOString() || "",
    }));
  },

  async getBlueprint(id: string): Promise<BlueprintUI | null> {
    const response = await catalogRpc.getBlueprint(id);
    if (!response) return null;

    return mapTransportBlueprintToUI(response);
  },

  async createBlueprint(
    modelId: string,
    data: CreateBlueprintRequest,
  ): Promise<BlueprintUI> {
    const response = await catalogRpc.createBlueprint(modelId, {
      name: data.version,
      version: data.version,
      decoderScript: JSON.stringify(data.specJson),
      isSystem: data.isSystem,
    });

    return mapTransportBlueprintToUI(response);
  },

  async updateBlueprint(
    id: string,
    data: UpdateBlueprintRequest,
  ): Promise<void> {
    await catalogRpc.updateBlueprint(id, {
      version: data.version,
      decoderScript: data.specJson ? JSON.stringify(data.specJson) : undefined,
    });
  },

  async setBlueprintDefault(id: string): Promise<void> {
    await catalogRpc.setDefaultBlueprint(id);
  },

  async countModelSnapshotEndpoints(deviceModelId: string): Promise<number> {
    return catalogRpc.countEndpointsByDeviceModel(deviceModelId);
  },

  // setAsDefault also moves the model's default pointer.
  async bulkAssignBlueprint(data: {
    blueprintId: string;
    deviceModelId: string;
    setAsDefault: boolean;
  }): Promise<BulkAssignBlueprintResponse> {
    return catalogRpc.bulkAssignBlueprint(data);
  },

  async decodePreview(
    id: string,
    data: DecodePreviewRequest,
  ): Promise<DecodePreviewResponse> {
    const payloadBytes = parseDecodePreviewPayload(data.userData);
    const result = await catalogRpc.decodePreview(
      id,
      payloadBytes,
      data.formatId || 0,
    );

    return {
      success: result.success,
      decodedData: result.decodedPayload,
      errorCode: result.errorCode,
      errorDetail: result.errorDetail,
      formatId: result.formatId,
    };
  },

  async submitToRegistry(
    id: string,
    data: RegistrySubmitRequest,
  ): Promise<RegistrySubmitResponse> {
    const response = await catalogRpc.submitBlueprintToRegistry(id, {
      contributorName: data.contributorName,
      contributorEmail: data.contributorEmail,
      notes: data.description,
    });

    return {
      prUrl: response.prUrl,
      commitSha: response.commitSha || "",
      branch: response.branchName || REGISTRY_DEFAULTS.BRANCH,
    };
  },
};
