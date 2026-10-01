/**
 * Tells the operator the service center has no URL base stations can reach
 * and names the setting that provides one.
 */

import { Alert } from "@mui/material";

import { SERVICE_CENTER_URL_SETTING } from "@constants/app";
import {
  WARN_SC_URL_ENV_ALTERNATIVE,
  WARN_SC_URL_NOT_CONFIGURED,
} from "@constants/messages";

export default function ServiceCenterUrlWarning({ mb = 3 }: { mb?: number }) {
  return (
    <Alert severity="warning" sx={{ mb }}>
      {WARN_SC_URL_NOT_CONFIGURED}{" "}
      <code>{SERVICE_CENTER_URL_SETTING.CONFIG_KEY}</code>{" "}
      {WARN_SC_URL_ENV_ALTERNATIVE}{" "}
      <code>{SERVICE_CENTER_URL_SETTING.ENV_VAR}</code>
    </Alert>
  );
}
