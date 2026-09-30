import { useState } from "react";

import type { BlueprintUI } from "@api-types/api";

import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { JSON_PREVIEW } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

import { specJsonError, validateBlueprintSpecJson } from "../utils/spec";
import { useUpdateBlueprint } from "./useBlueprints";

export function useBlueprintSpecEditor(
  blueprint: BlueprintUI | null | undefined,
) {
  const [isEditing, setIsEditing] = useState(false);
  const [spec, setSpec] = useState("");
  const [version, setVersion] = useState("");
  const [error, setError] = useState<string | null>(null);
  const update = useUpdateBlueprint();
  const feedback = useFeedback();

  const start = () => {
    if (!blueprint) return;
    setVersion(blueprint.version);
    setSpec(JSON.stringify(blueprint.specJson, null, JSON_PREVIEW.INDENT));
    setIsEditing(true);
    setError(null);
  };

  const cancel = () => {
    setIsEditing(false);
    setError(null);
  };

  const save = () => {
    if (!blueprint) return;
    const specError = specJsonError(spec);
    if (specError) {
      setError(specError);
      return;
    }
    const specJson = validateBlueprintSpecJson(spec);
    update.mutate(
      {
        id: blueprint.id,
        data: { version: version || undefined, specJson },
      },
      {
        onSuccess: () => {
          setIsEditing(false);
          setError(null);
          feedback.success(BLUEPRINT_LABELS.MSG_BLUEPRINT_UPDATED);
        },
        onError: (err: Error) => setError(getErrorMessage(err)),
      },
    );
  };

  return {
    isEditing,
    spec,
    setSpec,
    version,
    setVersion,
    error,
    start,
    save,
    cancel,
    isSaving: update.isPending,
  };
}
