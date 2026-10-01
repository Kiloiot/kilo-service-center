/**
 * Certificate RPCs (CoreService): base-station and server certificates.
 */

import type {
  CertificateSummary,
  ServerCertificateStatus,
} from "@api-types/api";

import * as corePb from "@services/grpc/core_pb";
import type { CertificateDownloadType } from "@constants/app";

import { grpcTransport } from "../transport";

/**
 * Generate certificate for base station
 */
export async function generateCertificate(data: {
  bsEui: string;
  baseStationName?: string;
  validityDays?: number;
}): Promise<{
  bsEui: string;
  serviceCenterUrl: string;
  downloadUrls: Record<string, string>;
  expiresAt: Date;
}> {
  const request = new corePb.GenerateCertificateRequest();
  request.setBsEui(data.bsEui);
  if (data.baseStationName) request.setBaseStationName(data.baseStationName);
  if (data.validityDays) request.setValidityDays(data.validityDays);

  const response = await grpcTransport.callCore<
    corePb.GenerateCertificateRequest,
    corePb.GenerateCertificateResponse
  >((c) => c.generateCertificate, request);

  // Convert downloadUrlsMap to plain object
  const downloadUrls: Record<string, string> = {};
  response.getDownloadUrlsMap().forEach((value, key) => {
    downloadUrls[key] = value;
  });

  return {
    bsEui: response.getBsEui(),
    serviceCenterUrl: response.getServiceCenterUrl(),
    downloadUrls,
    expiresAt: response.getExpiresAt()?.toDate() || new Date(),
  };
}

/**
 * Download certificate by ID and type
 */
export async function downloadCertificate(
  certId: string,
  certType: CertificateDownloadType,
): Promise<{
  content: Uint8Array;
  filename: string;
  contentType: string;
}> {
  const request = new corePb.DownloadCertificateRequest();
  request.setId(certId);
  request.setCertType(certType);

  const response = await grpcTransport.callCore<
    corePb.DownloadCertificateRequest,
    corePb.DownloadCertificateResponse
  >((c) => c.downloadCertificate, request);

  return {
    content: response.getContent_asU8(),
    filename: response.getFilename(),
    contentType: response.getContentType(),
  };
}

/**
 * Download a certificate stored on a base station record by its EUI.
 */
export async function downloadBaseStationCertificate(
  bsEui: string,
  certType: CertificateDownloadType,
): Promise<{
  content: Uint8Array;
  filename: string;
  contentType: string;
}> {
  const request = new corePb.DownloadBaseStationCertificateRequest();
  request.setBsEui(bsEui);
  request.setCertType(certType);

  const response = await grpcTransport.callCore<
    corePb.DownloadBaseStationCertificateRequest,
    corePb.DownloadCertificateResponse
  >((c) => c.downloadBaseStationCertificate, request);

  return {
    content: response.getContent_asU8(),
    filename: response.getFilename(),
    contentType: response.getContentType(),
  };
}

/**
 * Generate server certificates
 */
export async function generateServerCertificates(): Promise<{
  success: boolean;
  message: string;
}> {
  const request = new corePb.GenerateServerCertificatesRequest();

  const response = await grpcTransport.callCore<
    corePb.GenerateServerCertificatesRequest,
    corePb.GenerateServerCertificatesResponse
  >((c) => c.generateServerCertificates, request);

  return {
    success: response.getSuccess(),
    message: response.getMessage(),
  };
}

/**
 * Get server certificate status
 */
export async function getServerCertificateStatus(): Promise<ServerCertificateStatus> {
  const request = new corePb.GetServerCertificateStatusRequest();

  const response = await grpcTransport.callCore<
    corePb.GetServerCertificateStatusRequest,
    corePb.GetServerCertificateStatusResponse
  >((c) => c.getServerCertificateStatus, request);

  return {
    serverCert: toCertificateSummary(response.getServerCert()),
    caCert: toCertificateSummary(response.getCaCert()),
    renewalNames: response.getRenewalNamesList(),
  };
}

function toCertificateSummary(
  cert: corePb.CertificateStatus | undefined,
): CertificateSummary | undefined {
  if (!cert) return undefined;
  return {
    subject: cert.getSubject(),
    issuer: cert.getIssuer(),
    notBefore: cert.getNotBefore()?.toDate() || new Date(),
    notAfter: cert.getNotAfter()?.toDate() || new Date(),
    daysUntilExpiry: cert.getDaysUntilExpiry(),
    isValid: cert.getIsValid(),
  };
}

/**
 * Renew server certificates
 */
export async function renewServerCertificates(): Promise<{
  success: boolean;
  message: string;
}> {
  const request = new corePb.RenewServerCertificatesRequest();

  const response = await grpcTransport.callCore<
    corePb.RenewServerCertificatesRequest,
    corePb.RenewServerCertificatesResponse
  >((c) => c.renewServerCertificates, request);

  return {
    success: response.getSuccess(),
    message: response.getMessage(),
  };
}
