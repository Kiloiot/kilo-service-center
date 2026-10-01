/**
 * Why a base station certificate download failed, as the certificate card
 * shows it.
 */

import { isApiError } from "@api-types/api";

import { getErrorMessage } from "@utils/error-message";
import { HTTP_STATUS } from "@constants/app";
import { BS_OPERATIONS } from "@constants/messages";

/**
 * The service center answers a client certificate it holds no copy of as a
 * failed precondition, which regenerating the certificates resolves.
 */
export function certificateDownloadFailure(error: unknown): string {
  return isApiError(error) && error.status === HTTP_STATUS.PRECONDITION_FAILED
    ? BS_OPERATIONS.ERR_DOWNLOAD_CLIENT_NOT_STORED
    : getErrorMessage(error, BS_OPERATIONS.ERR_DOWNLOAD);
}
