/**
 * A session key field of the End Point form. A key the service has stored is
 * never sent with the endpoint: the field stays empty and masked until an
 * endpoint manager reveals it, which the service records, and it can be
 * copied only once revealed. A typed or generated key replaces it; a key that
 * may be removed offers a confirmed Remove that takes effect on save.
 */

import React, { useState } from "react";

import { Button, IconButton, InputAdornment, Tooltip } from "@mui/material";
import { ConfirmDialog } from "@ui";

import { useClipboard } from "@hooks/useClipboard";
import { ENDPOINT_FORM } from "@constants/messages";
import {
  CheckCircleIcon,
  ContentCopyIcon,
  VisibilityIcon,
  VisibilityOffIcon,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";

import type { EndpointFormState } from "../hooks";
import type { EndpointField } from "../utils/endpoint-validation";
import { EndpointTextField } from "./EndpointTextField";

interface SecretKeyFieldProps {
  form: EndpointFormState;
  field: EndpointField;
  label: string;
  helper: string;
  onGenerate: () => void;
  /** Reveals the stored key; absent when the endpoint has none stored. */
  onReveal?: () => Promise<string | undefined>;
  /** Marks the stored key for removal on save; absent when it cannot be removed. */
  onRemove?: () => void;
  /** The stored key is marked for removal. */
  removed?: boolean;
}

export const SecretKeyField: React.FC<SecretKeyFieldProps> = ({
  form,
  field,
  label,
  helper,
  onGenerate,
  onReveal,
  onRemove,
  removed = false,
}) => {
  const [shown, setShown] = useState(false);
  const [confirmingRemoval, setConfirmingRemoval] = useState(false);
  const [revealed, setRevealed] = useState<string>();
  const [revealing, setRevealing] = useState(false);
  const clipboard = useClipboard();

  const typed = String(form.values[field]);
  const showsStored = !typed && revealed !== undefined;
  const displayed = typed || (showsStored ? revealed : "");
  const storedAndHidden = !typed && !!onReveal && revealed === undefined;
  const copied = clipboard.copiedField === field;

  const toggle = async () => {
    if (shown || !storedAndHidden || !onReveal) {
      // Hiding forgets a revealed key, so it is fetched (and recorded) again next time.
      if (shown) setRevealed(undefined);
      setShown((current) => !current);
      return;
    }
    setRevealing(true);
    try {
      setRevealed((await onReveal()) ?? "");
      setShown(true);
    } catch {
      form.setGeneralError(ENDPOINT_FORM.ERROR_REVEAL_KEY);
    } finally {
      setRevealing(false);
    }
  };

  const confirmRemoval = () => {
    setShown(false);
    setRevealed(undefined);
    setConfirmingRemoval(false);
    onRemove?.();
  };

  const placeholder =
    removed && !typed
      ? ENDPOINT_FORM.PLACEHOLDER_KEY_REMOVED
      : storedAndHidden
        ? ENDPOINT_FORM.PLACEHOLDER_KEY_STORED
        : undefined;
  const toggleLabel = shown
    ? ENDPOINT_FORM.ACTION_HIDE_KEY
    : storedAndHidden
      ? ENDPOINT_FORM.ACTION_REVEAL_KEY
      : ENDPOINT_FORM.ACTION_SHOW_KEY;
  const copyLabel = copied
    ? ENDPOINT_FORM.LABEL_KEY_COPIED
    : ENDPOINT_FORM.ACTION_COPY_KEY;

  return (
    <>
      <EndpointTextField
        form={form}
        field={field}
        label={label}
        helper={helper}
        size={componentSpacing.gridSpan.full}
        value={displayed}
        placeholder={placeholder}
        type={shown ? "text" : "password"}
        inputProps={{ autoComplete: "off", spellCheck: false }}
        InputProps={{
          readOnly: showsStored,
          endAdornment: (
            <InputAdornment position="end">
              <Tooltip title={toggleLabel}>
                <span>
                  <IconButton
                    size="small"
                    aria-label={toggleLabel}
                    disabled={revealing}
                    onClick={toggle}
                  >
                    {shown ? (
                      <VisibilityOffIcon fontSize="small" />
                    ) : (
                      <VisibilityIcon fontSize="small" />
                    )}
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title={copyLabel}>
                <span>
                  <IconButton
                    size="small"
                    aria-label={copyLabel}
                    disabled={!displayed}
                    onClick={() => clipboard.copy(displayed, field)}
                  >
                    {copied ? (
                      <CheckCircleIcon fontSize="small" color="success" />
                    ) : (
                      <ContentCopyIcon fontSize="small" />
                    )}
                  </IconButton>
                </span>
              </Tooltip>
              <Button size="small" onClick={onGenerate}>
                {ENDPOINT_FORM.BUTTON_GENERATE}
              </Button>
              {onRemove && (
                <Button
                  size="small"
                  color="error"
                  onClick={() => setConfirmingRemoval(true)}
                >
                  {ENDPOINT_FORM.ACTION_REMOVE_KEY}
                </Button>
              )}
            </InputAdornment>
          ),
        }}
      />
      {onRemove && (
        <ConfirmDialog
          open={confirmingRemoval}
          title={ENDPOINT_FORM.DIALOG_REMOVE_KEY_TITLE}
          message={ENDPOINT_FORM.DIALOG_REMOVE_KEY_MESSAGE}
          confirmLabel={ENDPOINT_FORM.BUTTON_REMOVE_KEY}
          onConfirm={confirmRemoval}
          onClose={() => setConfirmingRemoval(false)}
        />
      )}
    </>
  );
};
