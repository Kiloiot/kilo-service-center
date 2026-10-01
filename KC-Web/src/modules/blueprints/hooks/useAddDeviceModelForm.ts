/**
 * Add Model form state: the catalog and manufacturer come from the page's
 * query, a successful create returns to that catalog.
 */

import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { BLUEPRINT_QUERY_PARAMS, BLUEPRINT_SCOPE } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

import { catalogPath, scopeFromQuery } from "../utils/catalog-routes";
import { specJsonError, validateBlueprintSpecJson } from "../utils/spec";
import { useCreateDeviceModelWithBlueprint } from "./useDeviceModels";

interface ModelFields {
  manufacturerId: string;
  name: string;
  version: string;
}

export type ModelFieldErrors = Partial<
  Record<keyof ModelFields | "specJson", string>
>;

const REQUIRED_FIELDS: ReadonlyArray<[keyof ModelFields, string]> = [
  ["manufacturerId", BLUEPRINT_LABELS.ERR_MANUFACTURER_REQUIRED],
  ["name", BLUEPRINT_LABELS.ERR_NAME_REQUIRED],
  ["version", BLUEPRINT_LABELS.ERR_VERSION_REQUIRED],
];

/** Every field left empty and a missing or invalid specification, as field messages. */
function modelFieldErrors(
  fields: ModelFields,
  specJsonText: string,
): ModelFieldErrors {
  const errors: ModelFieldErrors = {};
  for (const [field, message] of REQUIRED_FIELDS) {
    if (!fields[field].trim()) errors[field] = message;
  }
  const specError = specJsonError(specJsonText);
  if (specError) errors.specJson = specError;
  return errors;
}

export function useAddDeviceModelForm() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const scope = scopeFromQuery(searchParams);
  const [fields, setFields] = useState<ModelFields>({
    manufacturerId: searchParams.get(BLUEPRINT_QUERY_PARAMS.MANUFACTURER) ?? "",
    name: "",
    version: "",
  });
  const [specJsonText, setSpecJsonText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<ModelFieldErrors>({});
  const mutation = useCreateDeviceModelWithBlueprint();
  const feedback = useFeedback();

  const setField = (field: keyof ModelFields) => (value: string) => {
    setFields((current) => ({ ...current, [field]: value }));
    setFieldErrors((current) => ({ ...current, [field]: undefined }));
  };

  const editSpec = (text: string) => {
    setSpecJsonText(text);
    setFieldErrors((current) => ({ ...current, specJson: undefined }));
  };

  const submit = () => {
    const errors = modelFieldErrors(fields, specJsonText);
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;
    const specJson = validateBlueprintSpecJson(specJsonText);
    setError(null);
    mutation.mutate(
      { ...fields, specJson, isSystem: scope === BLUEPRINT_SCOPE.SYSTEM },
      {
        onSuccess: () => {
          navigate(catalogPath(scope));
          feedback.success(BLUEPRINT_LABELS.MSG_MODEL_CREATED);
        },
        onError: (err: Error) => setError(getErrorMessage(err)),
      },
    );
  };

  return {
    scope,
    fields,
    setField,
    specJsonText,
    setSpecJsonText: editSpec,
    error,
    fieldErrors,
    submit,
    cancel: () => navigate(catalogPath(scope)),
    isPending: mutation.isPending,
  };
}
