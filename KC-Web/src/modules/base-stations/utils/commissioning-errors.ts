import { isApiError } from "@api-types/api";

import { getErrorMessage, isNetworkError } from "@utils/error-message";
import { HTTP_STATUS } from "@constants/app";
import {
  ERR_BS_EUI_EXISTS,
  ERR_CERT_GENERATION_FAILED_RETRY,
  ERR_CERT_NETWORK,
  ERR_CREATE_BASE_STATION,
  ERR_SERVICE_UNREACHABLE,
} from "@constants/messages";

function mapCommissioningFailure(err: unknown, fallback: string): string {
  if (isNetworkError(err)) return ERR_CERT_NETWORK;
  if (isApiError(err) && err.status === HTTP_STATUS.SERVICE_UNAVAILABLE) {
    return ERR_SERVICE_UNREACHABLE;
  }
  return getErrorMessage(err, fallback);
}

/** Failure of the first step: creating the base station record. */
export function mapCommissionError(err: unknown): string {
  if (isApiError(err) && err.isAlreadyExists()) return ERR_BS_EUI_EXISTS;
  return mapCommissioningFailure(err, ERR_CREATE_BASE_STATION);
}

/** Failure of certificate generation for a base station that already exists. */
export function mapRetryCertsError(err: unknown): string {
  return mapCommissioningFailure(err, ERR_CERT_GENERATION_FAILED_RETRY);
}
