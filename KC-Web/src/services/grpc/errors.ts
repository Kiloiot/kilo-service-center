/**
 * gRPC-web error model: the HTTP-like status mapping and the error type the
 * rest of the application matches on.
 */

import { grpc } from "@improbable-eng/grpc-web";

import type { ServiceError } from "@services/grpc/core_pb_service";
import { HTTP_STATUS } from "@constants/app";

/** gRPC status codes mapped to the HTTP statuses the UI reasons about. */
export const GRPC_TO_HTTP_STATUS: Record<number, number> = {
  [grpc.Code.OK]: HTTP_STATUS.OK,
  [grpc.Code.Canceled]: HTTP_STATUS.CLIENT_CLOSED_REQUEST,
  [grpc.Code.Unknown]: HTTP_STATUS.INTERNAL_SERVER_ERROR,
  [grpc.Code.InvalidArgument]: HTTP_STATUS.BAD_REQUEST,
  [grpc.Code.DeadlineExceeded]: HTTP_STATUS.GATEWAY_TIMEOUT,
  [grpc.Code.NotFound]: HTTP_STATUS.NOT_FOUND,
  [grpc.Code.AlreadyExists]: HTTP_STATUS.CONFLICT,
  [grpc.Code.PermissionDenied]: HTTP_STATUS.FORBIDDEN,
  [grpc.Code.ResourceExhausted]: HTTP_STATUS.TOO_MANY_REQUESTS,
  [grpc.Code.FailedPrecondition]: HTTP_STATUS.PRECONDITION_FAILED,
  [grpc.Code.Aborted]: HTTP_STATUS.CONFLICT,
  [grpc.Code.OutOfRange]: HTTP_STATUS.BAD_REQUEST,
  [grpc.Code.Unimplemented]: HTTP_STATUS.NOT_IMPLEMENTED,
  [grpc.Code.Internal]: HTTP_STATUS.INTERNAL_SERVER_ERROR,
  [grpc.Code.Unavailable]: HTTP_STATUS.SERVICE_UNAVAILABLE,
  [grpc.Code.DataLoss]: HTTP_STATUS.INTERNAL_SERVER_ERROR,
  [grpc.Code.Unauthenticated]: HTTP_STATUS.UNAUTHORIZED,
};

/** Error catalog tokens the backend embeds in status messages. */
const ERROR_TOKEN_PATTERN = /KC-GRPC-ERR-\w+/;

/**
 * Transport error carried to hooks and components; matches the
 * ApiErrorLike contract in @api-types/api.
 */
export class GrpcApiError extends Error {
  readonly status: number;
  readonly details?: unknown;
  readonly code?: string;
  readonly token?: string;
  readonly grpcCode?: number;

  constructor(
    status: number,
    message: string,
    details?: unknown,
    code?: string,
    token?: string,
    grpcCode?: number,
  ) {
    super(message);
    this.name = "GrpcApiError";
    this.status = status;
    this.details = details;
    this.code = code;
    this.token = token;
    this.grpcCode = grpcCode;
  }

  isNotFound(): boolean {
    return (
      this.status === HTTP_STATUS.NOT_FOUND ||
      this.grpcCode === grpc.Code.NotFound
    );
  }

  isUnauthorized(): boolean {
    return (
      this.status === HTTP_STATUS.UNAUTHORIZED ||
      this.grpcCode === grpc.Code.Unauthenticated
    );
  }

  isForbidden(): boolean {
    return (
      this.status === HTTP_STATUS.FORBIDDEN ||
      this.grpcCode === grpc.Code.PermissionDenied
    );
  }

  isAlreadyExists(): boolean {
    return this.grpcCode === grpc.Code.AlreadyExists;
  }

  isInvalidArgument(): boolean {
    return (
      this.status === HTTP_STATUS.BAD_REQUEST ||
      this.grpcCode === grpc.Code.InvalidArgument
    );
  }
}

/** Converts a transport ServiceError into the application error type. */
export function toGrpcApiError(error: ServiceError): GrpcApiError {
  const httpStatus =
    GRPC_TO_HTTP_STATUS[error.code] || HTTP_STATUS.INTERNAL_SERVER_ERROR;
  const tokenMatch = error.message?.match(ERROR_TOKEN_PATTERN);
  const token = tokenMatch ? tokenMatch[0] : undefined;
  return new GrpcApiError(
    httpStatus,
    error.message,
    undefined,
    undefined,
    token,
    error.code,
  );
}
