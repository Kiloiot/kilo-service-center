/**
 * A new password and its confirmation: the password field states the rule
 * the identity service publishes, and the confirmation flags a mismatch as
 * soon as it is typed.
 */

import React from "react";

import { usePasswordRule } from "@hooks";
import { TextField } from "@mui/material";

import { PASSWORD_RULES } from "@constants/messages";

interface NewPasswordFieldsProps {
  password: string;
  confirmation: string;
  onPasswordChange: (value: string) => void;
  onConfirmationChange: (value: string) => void;
  passwordLabel: string;
  confirmationLabel: string;
  passwordError?: string;
  confirmationError?: string;
  disabled?: boolean;
  autoFocus?: boolean;
  required?: boolean;
}

/** The confirmation differs from the password once one was typed. */
function passwordsDiffer(password: string, confirmation: string) {
  return confirmation !== "" && password !== confirmation;
}

export const NewPasswordFields: React.FC<NewPasswordFieldsProps> = ({
  password,
  confirmation,
  onPasswordChange,
  onConfirmationChange,
  passwordLabel,
  confirmationLabel,
  passwordError,
  confirmationError,
  disabled,
  autoFocus,
  required,
}) => {
  const rule = usePasswordRule();
  const confirmationHelp =
    confirmationError ||
    (passwordsDiffer(password, confirmation)
      ? PASSWORD_RULES.ERR_MISMATCH
      : undefined);
  return (
    <>
      <TextField
        label={passwordLabel}
        type="password"
        value={password}
        onChange={(e) => onPasswordChange(e.target.value)}
        error={!!passwordError}
        helperText={passwordError || rule}
        disabled={disabled}
        autoFocus={autoFocus}
        required={required}
        fullWidth
      />
      <TextField
        label={confirmationLabel}
        type="password"
        value={confirmation}
        onChange={(e) => onConfirmationChange(e.target.value)}
        error={!!confirmationHelp}
        helperText={confirmationHelp}
        disabled={disabled}
        required={required}
        fullWidth
      />
    </>
  );
};
