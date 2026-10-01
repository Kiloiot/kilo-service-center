/**
 * The manufacturer and model names of an end point's blueprint; when the
 * catalog cannot be read the device model id stands in for both.
 */

import { useMemo } from "react";

import { useDeviceModel, useManufacturer } from "@modules/blueprints/hooks";

import type { BlueprintInfo } from "../components/EndpointBlueprintSection";

export function useBlueprintInfo(
  deviceModelId: string | undefined,
): BlueprintInfo | null {
  const modelQuery = useDeviceModel(deviceModelId);
  const manufacturerQuery = useManufacturer(modelQuery.data?.manufacturerId);
  return useMemo(() => {
    if (!deviceModelId) return null;
    if (modelQuery.isError || manufacturerQuery.isError) {
      return { manufacturerName: deviceModelId, modelName: deviceModelId };
    }
    if (!modelQuery.data) return null;
    return {
      manufacturerName: manufacturerQuery.data?.name || deviceModelId,
      modelName: modelQuery.data.name,
    };
  }, [
    deviceModelId,
    modelQuery.data,
    modelQuery.isError,
    manufacturerQuery.data,
    manufacturerQuery.isError,
  ]);
}
