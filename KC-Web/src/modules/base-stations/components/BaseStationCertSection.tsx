/**
 * Certificate section within the base station edit dialog.
 * Shows either regenerate button or post-regen download buttons.
 */

import React from "react";

import type { GenerateCertificateResponse } from "@api-types/api";
import { Alert, Box, Button, Divider, Typography } from "@mui/material";

import { apiService } from "@services/api";
import { getMonoBody1 } from "@utils/typography";
import { BASE_STATION_DETAILS } from "@constants/messages";
import { DownloadIcon } from "@theme/icons";

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
  const handleDownloadCert = async (certType: "ca" | "client" | "key") => {
    if (!regenCertData?.downloadUrls) return;

    const certIdMap: Record<"ca" | "client" | "key", string | undefined> = {
      ca: regenCertData.downloadUrls.caCert,
      client: regenCertData.downloadUrls.clientCert,
      key: regenCertData.downloadUrls.privateKey,
    };

    const certId = certIdMap[certType];
    if (!certId) return;

    try {
      const { blob, filename } = await apiService.downloadCertificate(
        certId,
        certType,
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      onError(BASE_STATION_DETAILS.CERTIFICATE_DOWNLOAD_ERROR);
    }
  };

  return (
    <>
      <Divider sx={{ mb: 2 }} />
      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
        {BASE_STATION_DETAILS.CERTIFICATES_SECTION_TITLE}
      </Typography>

      {regenCertData ? (
        <Box>
          <Alert severity="success" sx={{ mb: 2 }}>
            {BASE_STATION_DETAILS.REGENERATE_CERTS_SUCCESS}
          </Alert>
          {effectiveServiceCenterUrl && (
            <ScUrlCopyField
              value={effectiveServiceCenterUrl}
              copied={scUrlCopied}
              onCopy={onCopyScUrl}
              inputSxGetter={getMonoBody1}
              mb={2}
            />
          )}
          <Box display="flex" gap={1} flexWrap="wrap">
            <Button
              variant="outlined"
              size="small"
              startIcon={<DownloadIcon />}
              onClick={() => handleDownloadCert("ca")}
            >
              {BASE_STATION_DETAILS.DOWNLOAD_CA}
            </Button>
            <Button
              variant="outlined"
              size="small"
              startIcon={<DownloadIcon />}
              onClick={() => handleDownloadCert("client")}
            >
              {BASE_STATION_DETAILS.DOWNLOAD_CERT}
            </Button>
            <Button
              variant="outlined"
              size="small"
              startIcon={<DownloadIcon />}
              onClick={() => handleDownloadCert("key")}
            >
              {BASE_STATION_DETAILS.DOWNLOAD_KEY}
            </Button>
          </Box>
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
