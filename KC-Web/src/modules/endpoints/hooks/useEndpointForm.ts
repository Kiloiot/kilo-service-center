/**
 * Endpoint form state shared by the create and edit dialogs: values, field
 * errors (each field validated once it was left, and on every edit after),
 * dirtiness against the loaded endpoint, and the request builders.
 */

import React, { useCallback, useMemo, useState } from "react";

import type { EndpointUI } from "@api-types/api";

import { formatEuiInput } from "@utils/eui";
import { generateRandomKey } from "@utils/formatters";
import { MIOTY_KEY_BYTE_LENGTH } from "@constants/app";

import {
  buildChangedEndpointRequest,
  buildCreateEndpointPayload,
  buildEditEndpointFormData,
  changedStationProfileFields,
  EMPTY_ENDPOINT_FORM,
  type EndpointFormValues,
} from "../utils/endpoint-form";
import {
  type EndpointField,
  validateEndpointField,
  validateEndpointForm,
} from "../utils/endpoint-validation";

/** How a typed value is shown in its field; unlisted fields keep the input. */
const INPUT_FORMATS: Partial<Record<EndpointField, (value: string) => string>> =
  {
    epEui: formatEuiInput,
  };

type InputEvent = React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>;

const ALL_FIELDS = Object.keys(EMPTY_ENDPOINT_FORM) as EndpointField[];

/** The key fields whose key the loaded endpoint has stored. */
function storedKeyFields(endpoint?: EndpointUI | null): Set<EndpointField> {
  const stored = new Set<EndpointField>();
  if (endpoint?.nwkSnKeySet) stored.add("networkKey");
  if (endpoint?.appKeySet) stored.add("applicationKey");
  return stored;
}

/** The errors of the fields the administrator has left, keyed by field. */
function touchedErrors(
  values: EndpointFormValues,
  touched: ReadonlySet<EndpointField>,
  storedKeys: ReadonlySet<EndpointField>,
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const field of touched) {
    const error = validateEndpointField(field, values, storedKeys);
    if (error) errors[field] = error;
  }
  return errors;
}

export function useEndpointForm() {
  const [values, setValues] = useState<EndpointFormValues>(EMPTY_ENDPOINT_FORM);
  const [original, setOriginal] = useState<EndpointFormValues | null>(null);
  const [touched, setTouched] = useState<ReadonlySet<EndpointField>>(new Set());
  const [generalError, setGeneralError] = useState("");
  const [storedKeys, setStoredKeys] =
    useState<ReadonlySet<EndpointField>>(storedKeyFields());
  const [storedEpEui, setStoredEpEui] = useState<string>();

  const errors = useMemo<Record<string, string>>(
    () => ({
      ...touchedErrors(values, touched, storedKeys),
      ...(generalError ? { general: generalError } : {}),
    }),
    [values, touched, storedKeys, generalError],
  );

  const setField = useCallback(
    <K extends EndpointField>(field: K, value: EndpointFormValues[K]) =>
      setValues((prev) => ({ ...prev, [field]: value })),
    [],
  );

  /** Change handler for text and checkbox inputs bound to a field. */
  const handleInput = useCallback(
    (field: EndpointField) => (event: InputEvent) => {
      const target = event.target as HTMLInputElement;
      const format = INPUT_FORMATS[field];
      const value =
        target.type === "checkbox"
          ? target.checked
          : (format?.(target.value) ?? target.value);
      setField(field, value as EndpointFormValues[typeof field]);
    },
    [setField],
  );

  /** Validates a field from the moment the administrator leaves it. */
  const handleBlur = useCallback(
    (field: EndpointField) => () =>
      setTouched((prev) => new Set(prev).add(field)),
    [],
  );

  // A device model carries its own type EUI, so picking one clears the manual value.
  const setDeviceModel = useCallback((id?: string) => {
    setValues((prev) => ({
      ...prev,
      deviceModelId: id || "",
      typeEui: id ? "" : prev.typeEui,
    }));
  }, []);

  const reset = useCallback((endpoint?: EndpointUI | null) => {
    const next = endpoint
      ? buildEditEndpointFormData(endpoint)
      : EMPTY_ENDPOINT_FORM;
    setValues(next);
    setOriginal(endpoint ? next : null);
    setStoredKeys(storedKeyFields(endpoint));
    setStoredEpEui(endpoint?.epEui);
    setTouched(new Set());
    setGeneralError("");
  }, []);

  /** Shows every field's error and reports whether the form is valid. */
  const validate = useCallback((): boolean => {
    setTouched(new Set(ALL_FIELDS));
    return Object.keys(validateEndpointForm(values, storedKeys)).length === 0;
  }, [values, storedKeys]);

  const generateNetworkKey = useCallback(
    () => setField("networkKey", generateRandomKey(MIOTY_KEY_BYTE_LENGTH)),
    [setField],
  );
  const generateApplicationKey = useCallback(
    () => setField("applicationKey", generateRandomKey(MIOTY_KEY_BYTE_LENGTH)),
    [setField],
  );
  const removeApplicationKey = useCallback(
    () =>
      setValues((prev) => ({
        ...prev,
        applicationKey: "",
        removeApplicationKey: true,
      })),
    [],
  );

  const isDirty = original
    ? Object.keys(buildChangedEndpointRequest(values, original)).length > 0
    : false;

  const buildCreateRequest = useCallback(
    () => buildCreateEndpointPayload(values),
    [values],
  );
  const buildUpdateRequest = useCallback(
    () => (original ? buildChangedEndpointRequest(values, original) : {}),
    [values, original],
  );
  const changedProfileFields = useCallback(
    () => (original ? changedStationProfileFields(values, original) : []),
    [values, original],
  );

  return {
    values,
    errors,
    setField,
    setGeneralError,
    handleInput,
    handleBlur,
    setDeviceModel,
    reset,
    validate,
    isDirty,
    storedKeys,
    storedEpEui,
    generateNetworkKey,
    generateApplicationKey,
    removeApplicationKey,
    buildCreateRequest,
    buildUpdateRequest,
    changedProfileFields,
  };
}

/** The endpoint form the dialogs and their field sections share. */
export type EndpointFormState = ReturnType<typeof useEndpointForm>;
