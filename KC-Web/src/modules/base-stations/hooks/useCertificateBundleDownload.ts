import { useDownloadCertificate } from "@hooks";

import { downloadBlob } from "@utils/downloadBlob";
import type { CertificateDownloadType } from "@constants/app";
import { BASE_STATION_DETAILS } from "@constants/messages";

/**
 * Downloads one file of an issued certificate bundle into the browser. All
 * three files share the bundle id the issuance returned; a failed download
 * is reported through onError.
 */
export function useCertificateBundleDownload(
  bundleId: string | undefined,
  onError: (message: string) => void,
) {
  const downloadCertificate = useDownloadCertificate();
  return async (certType: CertificateDownloadType) => {
    if (!bundleId) return;
    try {
      const { blob, filename } = await downloadCertificate.mutateAsync({
        certId: bundleId,
        certType,
      });
      downloadBlob(blob, filename);
    } catch {
      onError(BASE_STATION_DETAILS.CERTIFICATE_DOWNLOAD_ERROR);
    }
  };
}
