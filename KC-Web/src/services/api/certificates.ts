/**
 * Certificates in the shapes the UI consumes.
 */

import type {
  GenerateCertificateRequest,
  GenerateCertificateResponse,
  ServerCertificateStatus,
} from "@api-types/api";
import type { DownloadableFile } from "@api-types/files";

import * as certificatesRpc from "@services/grpc/rpc/certificates";
import { fileFromBytes } from "@utils/downloadBlob";
import type {
  CertificateDownloadType,
  PublicCertificateType,
} from "@constants/app";
import { CERTIFICATE_BUNDLE_KEY } from "@constants/app";

export const certificatesApi = {
  async generateCertificate(
    data: GenerateCertificateRequest,
  ): Promise<GenerateCertificateResponse> {
    const response = await certificatesRpc.generateCertificate({
      bsEui: data.bsEui,
      validityDays: data.validityDays,
    });

    return {
      bsEui: data.bsEui,
      serviceCenterUrl: response.serviceCenterUrl || "",
      downloadUrls: {
        caCert: response.downloadUrls?.[CERTIFICATE_BUNDLE_KEY.CA_CERT] || "",
        clientCert:
          response.downloadUrls?.[CERTIFICATE_BUNDLE_KEY.CLIENT_CERT] || "",
        privateKey:
          response.downloadUrls?.[CERTIFICATE_BUNDLE_KEY.PRIVATE_KEY] || "",
      },
      expiresAt: response.expiresAt?.toISOString() || "",
    };
  },

  /**
   * Download certificate as Blob for browser download
   */
  async downloadCertificate(
    certId: string,
    certType: CertificateDownloadType,
  ): Promise<DownloadableFile> {
    const response = await certificatesRpc.downloadCertificate(
      certId,
      certType,
    );
    return fileFromBytes(
      response.content,
      response.filename,
      response.contentType,
    );
  },

  /** A base station's public certificate: the service center CA or its stored client certificate; never the private key. */
  async downloadBaseStationCertificate(
    bsEui: string,
    certType: PublicCertificateType,
  ): Promise<DownloadableFile> {
    const response = await certificatesRpc.downloadBaseStationCertificate(
      bsEui,
      certType,
    );
    return fileFromBytes(
      response.content,
      response.filename,
      response.contentType,
    );
  },

  async getCertificateStatus(): Promise<ServerCertificateStatus> {
    return certificatesRpc.getServerCertificateStatus();
  },

  async generateServerCertificates(): Promise<{
    success: boolean;
    message: string;
  }> {
    return certificatesRpc.generateServerCertificates();
  },

  async renewServerCertificates(): Promise<{
    success: boolean;
    message: string;
  }> {
    return certificatesRpc.renewServerCertificates();
  },
};
