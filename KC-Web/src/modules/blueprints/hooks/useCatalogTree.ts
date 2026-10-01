/**
 * Expansion state of the catalog tree, kept in sync with the route: a
 * manufacturer or device model named in the URL is expanded on arrival.
 */

import { useCallback, useEffect, useState } from "react";

import { useDeviceModel } from "./useDeviceModels";

const toggled = (set: Set<string>, id: string): Set<string> => {
  const next = new Set(set);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  return next;
};

const withId = (set: Set<string>, id: string): Set<string> =>
  set.has(id) ? set : new Set(set).add(id);

export function useCatalogTree({
  mfrId,
  modelId,
}: {
  mfrId?: string;
  modelId?: string;
}) {
  const [expandedMfrs, setExpandedMfrs] = useState<Set<string>>(new Set());
  const [expandedModels, setExpandedModels] = useState<Set<string>>(new Set());
  const { data: routeModel } = useDeviceModel(modelId);

  const expandPath = useCallback((manufacturerId: string, id?: string) => {
    setExpandedMfrs((prev) => withId(prev, manufacturerId));
    if (id) setExpandedModels((prev) => withId(prev, id));
  }, []);

  useEffect(() => {
    if (mfrId) expandPath(mfrId);
  }, [mfrId, expandPath]);

  useEffect(() => {
    if (routeModel) expandPath(routeModel.manufacturerId, routeModel.id);
  }, [routeModel, expandPath]);

  const toggleMfr = useCallback(
    (id: string) => setExpandedMfrs((prev) => toggled(prev, id)),
    [],
  );
  const toggleModel = useCallback(
    (id: string) => setExpandedModels((prev) => toggled(prev, id)),
    [],
  );

  return { expandedMfrs, expandedModels, toggleMfr, toggleModel, expandPath };
}
