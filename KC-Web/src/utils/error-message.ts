/**
 * One place that turns a caught error into the message the UI shows.
 */

import { isApiError } from "@api-types/api";

import { BROWSER_NETWORK_FAILURE_MESSAGE, HTTP_STATUS } from "@constants/app";
import {
  ERR_FORBIDDEN,
  ERR_GENERIC_ERROR,
  ERR_NETWORK_ERROR,
  ERR_TIMEOUT_ERROR,
} from "@constants/messages";

/** The browser could not reach the gateway at all. */
export function isNetworkError(err: unknown): boolean {
  return (
    err instanceof TypeError && err.message === BROWSER_NETWORK_FAILURE_MESSAGE
  );
}

// Service catalogs phrase errors in lower case ("failed to create base station").
const asSentence = (message: string): string =>
  message.charAt(0).toUpperCase() + message.slice(1);

export function getErrorMessage(
  err: unknown,
  fallback: string = ERR_GENERIC_ERROR,
): string {
  if (isApiError(err)) {
    if (err.isForbidden()) return ERR_FORBIDDEN;
    if (err.status === HTTP_STATUS.GATEWAY_TIMEOUT) return ERR_TIMEOUT_ERROR;
    return err.message ? asSentence(err.message) : fallback;
  }
  if (isNetworkError(err)) return ERR_NETWORK_ERROR;
  if (err instanceof Error)
    return err.message ? asSentence(err.message) : fallback;
  return fallback;
}
