/**
 * Shared auth page branding block (logo, edition chip, documentation links).
 * Used by Login and Register pages.
 */

import React from "react";

import { Box, Chip, Link as MuiLink } from "@mui/material";

import { APP_EDITION, APP_NAME } from "@constants/app";
import { BRAND } from "@constants/messages";
import kiloLogo from "@assets/kilo-logo.png";

interface AuthBrandingProps {
  versionInfo?: {
    edition?: string;
    documentationUrl?: string;
    sourceUrl?: string;
    licenseUrl?: string;
  } | null;
  /** Bottom margin on the links row. Defaults to 3. */
  linksMb?: number;
}

const AuthBranding: React.FC<AuthBrandingProps> = ({
  versionInfo,
  linksMb = 3,
}) => (
  <Box
    sx={{
      mb: 3,
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
    }}
  >
    <img
      src={kiloLogo}
      alt={APP_NAME}
      style={{ maxWidth: "200px", height: "auto", marginBottom: "8px" }}
    />
    <Chip
      label={versionInfo?.edition ?? APP_EDITION}
      size="small"
      variant="outlined"
      sx={{ mb: 2 }}
    />
    <Box
      sx={{ display: "flex", gap: 2, justifyContent: "center", mb: linksMb }}
    >
      <MuiLink
        href={versionInfo?.documentationUrl}
        target="_blank"
        rel="noopener"
        variant="caption"
      >
        {BRAND.DOCUMENTATION}
      </MuiLink>
      <MuiLink
        href={versionInfo?.sourceUrl}
        target="_blank"
        rel="noopener"
        variant="caption"
      >
        {BRAND.SOURCE}
      </MuiLink>
      <MuiLink
        href={versionInfo?.licenseUrl}
        target="_blank"
        rel="noopener"
        variant="caption"
      >
        {BRAND.LICENSE}
      </MuiLink>
    </Box>
  </Box>
);

export default AuthBranding;
