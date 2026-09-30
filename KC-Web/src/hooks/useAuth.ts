/**
 * Authentication mutations: the session pages drive login, registration,
 * provider code exchange, profile load, logout and password change through
 * these instead of calling the api facade.
 */

import type {
  ExchangeRequest,
  LoginRequest,
  RegisterAccountRequest,
} from "@api-types/api";
import { useMutation } from "@tanstack/react-query";

import { authApi } from "@services/api";

export function useLogin() {
  return useMutation({
    mutationFn: (request: LoginRequest) => authApi.login(request),
  });
}

export function useRegisterAccount() {
  return useMutation({
    mutationFn: (request: RegisterAccountRequest) =>
      authApi.registerAccount(request),
  });
}

/** OIDC first; the backend rejects a state that belongs to the OAuth2 flow. */
export function useExchangeAuthCode() {
  return useMutation({
    mutationFn: async (request: ExchangeRequest) => {
      try {
        return await authApi.exchangeOIDC(request);
      } catch {
        return authApi.exchangeOAuth2(request);
      }
    },
  });
}

export function useLoadAuthProfile() {
  return useMutation({ mutationFn: () => authApi.getAuthProfile() });
}

export function useLogout() {
  return useMutation({
    mutationFn: (refreshToken: string) => authApi.logout(refreshToken),
  });
}

export function useChangeOwnPassword() {
  return useMutation({
    mutationFn: ({
      currentPassword,
      newPassword,
    }: {
      currentPassword: string;
      newPassword: string;
    }) => authApi.changeOwnPassword(currentPassword, newPassword),
  });
}
