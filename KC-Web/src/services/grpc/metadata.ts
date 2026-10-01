/**
 * Metadata every gRPC-web call and stream sends: the bearer token from
 * storage plus the organization and user the call is scoped to.
 */

import { grpc } from "@improbable-eng/grpc-web";

import { storageService } from "@utils/storage";
import { HEADER_VALUES, HEADERS, STORAGE_KEYS } from "@constants/app";

/** Organization and user the metadata is scoped to. */
export interface AuthContext {
  organizationId: string | null;
  userId: string | null;
}

export function buildAuthMetadata(context: AuthContext): grpc.Metadata {
  const metadata = new grpc.Metadata();
  const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
  if (token) {
    metadata.set(
      HEADERS.AUTHORIZATION.toLowerCase(),
      `${HEADER_VALUES.AUTH_BEARER_PREFIX}${token}`,
    );
  }
  if (context.organizationId) {
    metadata.set(
      HEADERS.X_ORGANIZATION_ID.toLowerCase(),
      context.organizationId,
    );
  }
  if (context.userId) {
    metadata.set(HEADERS.X_USER_ID.toLowerCase(), context.userId);
  }
  return metadata;
}
