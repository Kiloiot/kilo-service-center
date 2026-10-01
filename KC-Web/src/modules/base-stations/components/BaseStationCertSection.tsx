/**
 * Certificate section within the base station edit dialog: the regenerate
 * action, then the files of the new certificate.
 */

import React from "react";

import type { GenerateCertificateResponse } from "@api-types/api";
import { Box, Button, Divider, Typography } from "@mui/material";

import { getMonoBody1 } from "@utils/typography";
import { BASE_STATION_DETAILS } from "@constants/messages";

import { useCertificateBundleDownload } from "../hooks";
import CertificateDownloadButtons from "./CertificateDownloadButtons";
import ScUrlCopyField from "./ScUrlCopyField";

interface BaseStationCertSectionProps {
  regenCertData: GenerateCertificateResponse | null;
  effectiveServiceCenterUrl: string | undefined;
  scUrlCopied: boolean;
  onCopyScUrl: () => void;
  isRegenerating: boolean;
  onRegenerate: () => void;
  onError: (message: string) => void;
}

const BaseStationCertSection: React.FC<BaseStationCertSectionProps> = ({
  regenCertData,
  effectiveServiceCenterUrl,
  scUrlCopied,
  onCopyScUrl,
  isRegenerating,
  onRegenerate,
  onError,
}) => {
  const handleDownload = useCertificateBundleDownload(
    regenCertData?.downloadUrls.caCert,
    onError,
  );

  return (
    <>
      <Divider sx={{ mb: 2 }} />
      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
        {BASE_STATION_DETAILS.CERTIFICATES_SECTION_TITLE}
      </Typography>

      {regenCertData ? (
        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {BASE_STATION_DETAILS.REGENERATE_CERTS_INSTALL_HINT}
          </Typography>
          {effectiveServiceCenterUrl && (
            <ScUrlCopyField
              value={effectiveServiceCenterUrl}
              copied={scUrlCopied}
              onCopy={onCopyScUrl}
              inputSxGetter={getMonoBody1}
              mb={2}
            />
          )}
          <CertificateDownloadButtons onDownload={handleDownload} />
        </Box>
      ) : (
        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {BASE_STATION_DETAILS.CERTIFICATES_HINT}
          </Typography>
          <Button
            variant="outlined"
            color="warning"
            onClick={onRegenerate}
            disabled={isRegenerating}
          >
            {BASE_STATION_DETAILS.ACTION_REGENERATE_CERTS}
          </Button>
        </Box>
      )}
    </>
  );
};

export default BaseStationCertSection;
