import { Box, Button, Paper, Typography } from "@mui/material";

import type { CertificateDownloadType } from "@constants/app";
import { CERTIFICATE_DOWNLOAD_TYPES } from "@constants/app";
import {
  ACTION_DOWNLOAD_CA_CERT,
  ACTION_DOWNLOAD_TLS_CERT,
  ACTION_DOWNLOAD_TLS_KEY,
  INFO_CERT_EXPIRY,
  SECTION_DOWNLOAD_CERTS,
} from "@constants/messages";
import { DownloadIcon } from "@theme/icons";

const DOWNLOADS: {
  certType: CertificateDownloadType;
  label: string;
  color?: "warning";
}[] = [
  { certType: CERTIFICATE_DOWNLOAD_TYPES.CA, label: ACTION_DOWNLOAD_CA_CERT },
  {
    certType: CERTIFICATE_DOWNLOAD_TYPES.CLIENT,
    label: ACTION_DOWNLOAD_TLS_CERT,
  },
  {
    certType: CERTIFICATE_DOWNLOAD_TYPES.KEY,
    label: ACTION_DOWNLOAD_TLS_KEY,
    color: "warning",
  },
];

interface CertificateDownloadButtonsProps {
  onDownload: (certType: CertificateDownloadType) => void;
}

/** The three files of a freshly issued base station certificate bundle. */
export default function CertificateDownloadButtons({
  onDownload,
}: CertificateDownloadButtonsProps) {
  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
      <Typography variant="subtitle2" gutterBottom>
        {SECTION_DOWNLOAD_CERTS}
      </Typography>
      <Typography variant="caption" color="text.secondary" gutterBottom>
        {INFO_CERT_EXPIRY}
      </Typography>
      <Box sx={{ mt: 2, display: "flex", flexDirection: "column", gap: 1 }}>
        {DOWNLOADS.map((download) => (
          <Button
            key={download.certType}
            onClick={() => onDownload(download.certType)}
            startIcon={<DownloadIcon />}
            variant="outlined"
            size="small"
            color={download.color}
          >
            {download.label}
          </Button>
        ))}
      </Box>
    </Paper>
  );
}
