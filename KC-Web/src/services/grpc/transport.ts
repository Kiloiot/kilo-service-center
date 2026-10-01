/**
 * gRPC-web transport kernel: owns the generated service clients, the
 * organization/user context every call is scoped to, and the token renewal
 * every unary call and server stream shares: an access token that is expired
 * or about to expire is renewed before it is sent, and a call or stream the
 * server still finds unauthenticated is sent once more after a renewal. Only
 * a renewal the server refuses sends the user to sign-in; one the service
 * cannot answer fails the call as unavailable and keeps the session.
 */

import { grpc } from "@improbable-eng/grpc-web";

import type { ServiceError } from "@services/grpc/core_pb_service";
import * as CoreServiceModule from "@services/grpc/core_pb_service";
import * as identityPb from "@services/grpc/identity_pb";
import * as IdentityServiceModule from "@services/grpc/identity_pb_service";
import { redirectToSignIn } from "@utils/signInRedirect";
import { storageService } from "@utils/storage";
import { HTTP_STATUS, TOKEN_REFRESH_OUTCOME } from "@constants/app";
import {
  ERR_AUTH_ORG_REQUIRED,
  ERR_UNAUTHORIZED,
  GRPC_CLIENT_ERRORS,
} from "@constants/messages";
import { grpcUrl } from "@config/env";

import { GrpcApiError, toGrpcApiError } from "./errors";
import { type AuthContext, buildAuthMetadata } from "./metadata";
import {
  type ExclusiveLock,
  type RefreshOutcome,
  type RotatedTokens,
  TokenRefreshCoordinator,
} from "./refreshCoordinator";
import { openRenewingStream } from "./renewingStream";
import { webExclusiveLock } from "./webLock";

const { CoreServiceClient } = CoreServiceModule;
const { IdentityServiceClient } = IdentityServiceModule;

export type CoreClient = InstanceType<typeof CoreServiceClient>;
export type IdentityClient = InstanceType<typeof IdentityServiceClient>;

/** Callback-style unary method as generated for the grpc-web clients. */
export type UnaryMethod<TRequest, TResponse> = (
  request: TRequest,
  metadata: grpc.Metadata,
  callback: (error: ServiceError | null, response: TResponse | null) => void,
) => unknown;

export interface CallOptions {
  /** Fail closed without organization/user context (default true). */
  requireOrgUser?: boolean;
  /** Never renew the access token for this call (sign-in, the refresh itself). */
  skipTokenRefresh?: boolean;
}

/** Options of a server stream; the transport supplies its metadata. */
export type StreamOptions<
  TRequest extends grpc.ProtobufMessage,
  TResponse extends grpc.ProtobufMessage,
> = Omit<grpc.InvokeRpcOptions<TRequest, TResponse>, "metadata">;

/** Invoked when the server refuses to renew the session of a call. */
export type AuthFailureCallback = () => void;

export class GrpcTransport {
  readonly core: CoreClient;
  readonly identity: IdentityClient;
  private organizationId: string | null = null;
  private userId: string | null = null;
  private authFailureCallback?: AuthFailureCallback;
  private readonly tokenRefresh: TokenRefreshCoordinator;

  /** An empty host routes through the Vite proxy in development. */
  constructor(
    host: string = grpcUrl,
    lock: ExclusiveLock = webExclusiveLock(),
  ) {
    this.core = new CoreServiceClient(host);
    this.identity = new IdentityServiceClient(host);
    this.tokenRefresh = new TokenRefreshCoordinator(
      (refreshToken) => this.rotateRefreshToken(refreshToken),
      storageService,
      lock,
    );
  }

  clearOrganization(): void {
    this.organizationId = null;
    this.userId = null;
  }

  setOrganization(
    orgId: string | null | undefined,
    userId?: string | null | undefined,
  ): void {
    if (!orgId) {
      this.clearOrganization();
      return;
    }
    this.organizationId = orgId;
    this.userId = userId || null;
  }

  setAuthFailureCallback(callback: AuthFailureCallback | undefined): void {
    this.authFailureCallback = callback;
  }

  /** Renews the stored tokens, one refresh at a time across every tab. */
  refreshTokens(): Promise<RefreshOutcome> {
    return this.tokenRefresh.refresh();
  }

  callCore<TRequest, TResponse>(
    pick: (client: CoreClient) => UnaryMethod<TRequest, TResponse>,
    request: TRequest,
    options: CallOptions = {},
  ): Promise<TResponse> {
    return this.call(this.core, pick(this.core), request, options);
  }

  callIdentity<TRequest, TResponse>(
    pick: (client: IdentityClient) => UnaryMethod<TRequest, TResponse>,
    request: TRequest,
    options: CallOptions = {},
  ): Promise<TResponse> {
    return this.call(this.identity, pick(this.identity), request, options);
  }

  /**
   * Opens a server stream through the same token renewal as a unary call:
   * an expiring token is renewed before it is sent, and a stream the server
   * refuses as unauthenticated is renewed once and reopened.
   */
  openStream<
    TRequest extends grpc.ProtobufMessage,
    TResponse extends grpc.ProtobufMessage,
    M extends grpc.MethodDefinition<TRequest, TResponse>,
  >(
    method: M,
    options: StreamOptions<TRequest, TResponse>,
    context: AuthContext,
  ): grpc.Request {
    return openRenewingStream(
      this.tokenRefresh,
      (handlers) =>
        grpc.invoke(method, {
          ...options,
          ...handlers,
          metadata: buildAuthMetadata(context),
        }),
      options,
    );
  }

  /**
   * Context of one call. Client-side fail-closed errors throw here, before
   * the call is sent or its token renewed.
   */
  private contextFor(options: CallOptions): AuthContext {
    if (options.requireOrgUser ?? true) {
      if (!this.organizationId) {
        throw new GrpcApiError(HTTP_STATUS.UNAUTHORIZED, ERR_AUTH_ORG_REQUIRED);
      }
      if (!this.userId) {
        throw new GrpcApiError(HTTP_STATUS.UNAUTHORIZED, ERR_UNAUTHORIZED);
      }
    }
    return { organizationId: this.organizationId, userId: this.userId };
  }

  private async call<TRequest, TResponse>(
    client: object,
    method: UnaryMethod<TRequest, TResponse>,
    request: TRequest,
    options: CallOptions,
  ): Promise<TResponse> {
    const context = this.contextFor(options);
    if (
      !options.skipTokenRefresh &&
      this.tokenRefresh.isAccessTokenExpiring()
    ) {
      await this.renew(
        new GrpcApiError(HTTP_STATUS.UNAUTHORIZED, ERR_UNAUTHORIZED),
      );
    }
    try {
      return await this.execute(
        client,
        method,
        request,
        buildAuthMetadata(context),
      );
    } catch (err) {
      if (
        options.skipTokenRefresh ||
        !(err instanceof GrpcApiError) ||
        !err.isUnauthorized()
      ) {
        throw err;
      }
      await this.renew(err);
      return this.execute(client, method, request, buildAuthMetadata(context));
    }
  }

  /**
   * Resolves once the tokens are renewed. A refusal ends the session with
   * the given error; an unanswered renewal fails the call as unavailable.
   */
  private async renew(refusal: GrpcApiError): Promise<void> {
    switch (await this.refreshTokens()) {
      case TOKEN_REFRESH_OUTCOME.RENEWED:
        return;
      case TOKEN_REFRESH_OUTCOME.REFUSED:
        return this.failAuthentication(refusal);
      case TOKEN_REFRESH_OUTCOME.UNAVAILABLE:
        throw new GrpcApiError(
          HTTP_STATUS.SERVICE_UNAVAILABLE,
          GRPC_CLIENT_ERRORS.TOKEN_REFRESH_UNAVAILABLE,
          undefined,
          undefined,
          undefined,
          grpc.Code.Unavailable,
        );
    }
  }

  /** Ends a call whose session the server refused to renew by going to sign-in. */
  private failAuthentication(err: GrpcApiError): Promise<never> {
    this.authFailureCallback?.();
    if (!redirectToSignIn()) return Promise.reject(err);
    // Never resolves: the page is unloading, so no error toast should flash.
    return new Promise<never>(() => {});
  }

  private execute<TRequest, TResponse>(
    client: object,
    method: UnaryMethod<TRequest, TResponse>,
    request: TRequest,
    metadata: grpc.Metadata,
  ): Promise<TResponse> {
    return new Promise((resolve, reject) => {
      method.call(client, request, metadata, (error, response) => {
        if (error) {
          reject(toGrpcApiError(error));
        } else if (response) {
          resolve(response);
        } else {
          reject(
            new GrpcApiError(
              HTTP_STATUS.INTERNAL_SERVER_ERROR,
              GRPC_CLIENT_ERRORS.EMPTY_RESPONSE,
            ),
          );
        }
      });
    });
  }

  private async rotateRefreshToken(
    refreshToken: string,
  ): Promise<RotatedTokens> {
    const request = new identityPb.RefreshTokensRequest();
    request.setRefreshToken(refreshToken);
    const response = await this.callIdentity<
      identityPb.RefreshTokensRequest,
      identityPb.RefreshTokensResponse
    >((c) => c.refreshTokens, request, {
      requireOrgUser: false,
      skipTokenRefresh: true,
    });
    const tokens = response.getTokens();
    if (!tokens) {
      throw new GrpcApiError(
        HTTP_STATUS.INTERNAL_SERVER_ERROR,
        GRPC_CLIENT_ERRORS.INVALID_REFRESH_TOKENS_RESPONSE,
      );
    }
    return {
      accessToken: tokens.getAccessToken(),
      refreshToken: tokens.getRefreshToken(),
    };
  }
}

export const grpcTransport = new GrpcTransport();
