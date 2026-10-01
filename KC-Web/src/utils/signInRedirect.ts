import { ROUTES } from "@constants/app";

/**
 * Sends the browser to the sign-in page; false when there is no page to
 * leave, outside a browser or already on sign-in.
 */
export function redirectToSignIn(): boolean {
  if (
    typeof window === "undefined" ||
    window.location.pathname === ROUTES.LOGIN
  ) {
    return false;
  }
  window.location.href = ROUTES.LOGIN;
  return true;
}
