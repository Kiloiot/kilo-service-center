// package: kilocenter.api.v1
// file: identity.proto

import * as identity_pb from "./identity_pb";
import { grpc } from "@improbable-eng/grpc-web";

type IdentityServiceLogin = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.LoginRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type IdentityServiceRefreshTokens = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RefreshTokensRequest;
  readonly responseType: typeof identity_pb.RefreshTokensResponse;
};

type IdentityServiceGetProfile = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetProfileRequest;
  readonly responseType: typeof identity_pb.GetProfileResponse;
};

type IdentityServiceGetAuthSettings = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetAuthSettingsRequest;
  readonly responseType: typeof identity_pb.GetAuthSettingsResponse;
};

type IdentityServiceLogout = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.LogoutRequest;
  readonly responseType: typeof identity_pb.LogoutResponse;
};

type IdentityServiceChangePassword = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ChangePasswordRequest;
  readonly responseType: typeof identity_pb.ChangePasswordResponse;
};

type IdentityServiceExchangeOIDC = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ExchangeOIDCRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type IdentityServiceExchangeOAuth2 = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ExchangeOAuth2Request;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type IdentityServiceRegisterAccount = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RegisterAccountRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type IdentityServiceCreateUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateUserRequest;
  readonly responseType: typeof identity_pb.CreateUserResponse;
};

type IdentityServiceGetUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetUserRequest;
  readonly responseType: typeof identity_pb.GetUserResponse;
};

type IdentityServiceUpdateUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateUserRequest;
  readonly responseType: typeof identity_pb.UpdateUserResponse;
};

type IdentityServiceDeleteUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteUserRequest;
  readonly responseType: typeof identity_pb.DeleteUserResponse;
};

type IdentityServiceListUsers = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListUsersRequest;
  readonly responseType: typeof identity_pb.ListUsersResponse;
};

type IdentityServiceUpdateUserPassword = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateUserPasswordRequest;
  readonly responseType: typeof identity_pb.UpdateUserPasswordResponse;
};

type IdentityServiceCreateOrganization = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateOrganizationRequest;
  readonly responseType: typeof identity_pb.CreateOrganizationResponse;
};

type IdentityServiceGetOrganization = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetOrganizationRequest;
  readonly responseType: typeof identity_pb.GetOrganizationResponse;
};

type IdentityServiceUpdateOrganization = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateOrganizationRequest;
  readonly responseType: typeof identity_pb.UpdateOrganizationResponse;
};

type IdentityServiceDeleteOrganization = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteOrganizationRequest;
  readonly responseType: typeof identity_pb.DeleteOrganizationResponse;
};

type IdentityServiceListOrganizations = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListOrganizationsRequest;
  readonly responseType: typeof identity_pb.ListOrganizationsResponse;
};

type IdentityServiceAddOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.AddOrganizationUserRequest;
  readonly responseType: typeof identity_pb.AddOrganizationUserResponse;
};

type IdentityServiceGetOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetOrganizationUserRequest;
  readonly responseType: typeof identity_pb.GetOrganizationUserResponse;
};

type IdentityServiceUpdateOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateOrganizationUserRequest;
  readonly responseType: typeof identity_pb.UpdateOrganizationUserResponse;
};

type IdentityServiceRemoveOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RemoveOrganizationUserRequest;
  readonly responseType: typeof identity_pb.RemoveOrganizationUserResponse;
};

type IdentityServiceListOrganizationUsers = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListOrganizationUsersRequest;
  readonly responseType: typeof identity_pb.ListOrganizationUsersResponse;
};

type IdentityServiceListUserOrganizations = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListUserOrganizationsRequest;
  readonly responseType: typeof identity_pb.ListUserOrganizationsResponse;
};

type IdentityServiceCreateApiKey = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateApiKeyRequest;
  readonly responseType: typeof identity_pb.CreateApiKeyResponse;
};

type IdentityServiceGetApiKey = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetApiKeyRequest;
  readonly responseType: typeof identity_pb.GetApiKeyResponse;
};

type IdentityServiceDeleteApiKey = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteApiKeyRequest;
  readonly responseType: typeof identity_pb.DeleteApiKeyResponse;
};

type IdentityServiceListApiKeys = {
  readonly methodName: string;
  readonly service: typeof IdentityService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListApiKeysRequest;
  readonly responseType: typeof identity_pb.ListApiKeysResponse;
};

export class IdentityService {
  static readonly serviceName: string;
  static readonly Login: IdentityServiceLogin;
  static readonly RefreshTokens: IdentityServiceRefreshTokens;
  static readonly GetProfile: IdentityServiceGetProfile;
  static readonly GetAuthSettings: IdentityServiceGetAuthSettings;
  static readonly Logout: IdentityServiceLogout;
  static readonly ChangePassword: IdentityServiceChangePassword;
  static readonly ExchangeOIDC: IdentityServiceExchangeOIDC;
  static readonly ExchangeOAuth2: IdentityServiceExchangeOAuth2;
  static readonly RegisterAccount: IdentityServiceRegisterAccount;
  static readonly CreateUser: IdentityServiceCreateUser;
  static readonly GetUser: IdentityServiceGetUser;
  static readonly UpdateUser: IdentityServiceUpdateUser;
  static readonly DeleteUser: IdentityServiceDeleteUser;
  static readonly ListUsers: IdentityServiceListUsers;
  static readonly UpdateUserPassword: IdentityServiceUpdateUserPassword;
  static readonly CreateOrganization: IdentityServiceCreateOrganization;
  static readonly GetOrganization: IdentityServiceGetOrganization;
  static readonly UpdateOrganization: IdentityServiceUpdateOrganization;
  static readonly DeleteOrganization: IdentityServiceDeleteOrganization;
  static readonly ListOrganizations: IdentityServiceListOrganizations;
  static readonly AddOrganizationUser: IdentityServiceAddOrganizationUser;
  static readonly GetOrganizationUser: IdentityServiceGetOrganizationUser;
  static readonly UpdateOrganizationUser: IdentityServiceUpdateOrganizationUser;
  static readonly RemoveOrganizationUser: IdentityServiceRemoveOrganizationUser;
  static readonly ListOrganizationUsers: IdentityServiceListOrganizationUsers;
  static readonly ListUserOrganizations: IdentityServiceListUserOrganizations;
  static readonly CreateApiKey: IdentityServiceCreateApiKey;
  static readonly GetApiKey: IdentityServiceGetApiKey;
  static readonly DeleteApiKey: IdentityServiceDeleteApiKey;
  static readonly ListApiKeys: IdentityServiceListApiKeys;
}

export type ServiceError = {
  message: string;
  code: number;
  metadata: grpc.Metadata;
};
export type Status = { details: string; code: number; metadata: grpc.Metadata };

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: "data", handler: (message: T) => void): ResponseStream<T>;
  on(type: "end", handler: (status?: Status) => void): ResponseStream<T>;
  on(type: "status", handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: "end", handler: (status?: Status) => void): RequestStream<T>;
  on(type: "status", handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(
    type: "data",
    handler: (message: ResT) => void,
  ): BidirectionalStream<ReqT, ResT>;
  on(
    type: "end",
    handler: (status?: Status) => void,
  ): BidirectionalStream<ReqT, ResT>;
  on(
    type: "status",
    handler: (status: Status) => void,
  ): BidirectionalStream<ReqT, ResT>;
}

export class IdentityServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  login(
    requestMessage: identity_pb.LoginRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  login(
    requestMessage: identity_pb.LoginRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  refreshTokens(
    requestMessage: identity_pb.RefreshTokensRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.RefreshTokensResponse | null,
    ) => void,
  ): UnaryResponse;
  refreshTokens(
    requestMessage: identity_pb.RefreshTokensRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.RefreshTokensResponse | null,
    ) => void,
  ): UnaryResponse;
  getProfile(
    requestMessage: identity_pb.GetProfileRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetProfileResponse | null,
    ) => void,
  ): UnaryResponse;
  getProfile(
    requestMessage: identity_pb.GetProfileRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetProfileResponse | null,
    ) => void,
  ): UnaryResponse;
  getAuthSettings(
    requestMessage: identity_pb.GetAuthSettingsRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetAuthSettingsResponse | null,
    ) => void,
  ): UnaryResponse;
  getAuthSettings(
    requestMessage: identity_pb.GetAuthSettingsRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetAuthSettingsResponse | null,
    ) => void,
  ): UnaryResponse;
  logout(
    requestMessage: identity_pb.LogoutRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LogoutResponse | null,
    ) => void,
  ): UnaryResponse;
  logout(
    requestMessage: identity_pb.LogoutRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LogoutResponse | null,
    ) => void,
  ): UnaryResponse;
  changePassword(
    requestMessage: identity_pb.ChangePasswordRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ChangePasswordResponse | null,
    ) => void,
  ): UnaryResponse;
  changePassword(
    requestMessage: identity_pb.ChangePasswordRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ChangePasswordResponse | null,
    ) => void,
  ): UnaryResponse;
  exchangeOIDC(
    requestMessage: identity_pb.ExchangeOIDCRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  exchangeOIDC(
    requestMessage: identity_pb.ExchangeOIDCRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  exchangeOAuth2(
    requestMessage: identity_pb.ExchangeOAuth2Request,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  exchangeOAuth2(
    requestMessage: identity_pb.ExchangeOAuth2Request,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  registerAccount(
    requestMessage: identity_pb.RegisterAccountRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  registerAccount(
    requestMessage: identity_pb.RegisterAccountRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.LoginResponse | null,
    ) => void,
  ): UnaryResponse;
  createUser(
    requestMessage: identity_pb.CreateUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateUserResponse | null,
    ) => void,
  ): UnaryResponse;
  createUser(
    requestMessage: identity_pb.CreateUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateUserResponse | null,
    ) => void,
  ): UnaryResponse;
  getUser(
    requestMessage: identity_pb.GetUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetUserResponse | null,
    ) => void,
  ): UnaryResponse;
  getUser(
    requestMessage: identity_pb.GetUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetUserResponse | null,
    ) => void,
  ): UnaryResponse;
  updateUser(
    requestMessage: identity_pb.UpdateUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateUserResponse | null,
    ) => void,
  ): UnaryResponse;
  updateUser(
    requestMessage: identity_pb.UpdateUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateUserResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteUser(
    requestMessage: identity_pb.DeleteUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteUserResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteUser(
    requestMessage: identity_pb.DeleteUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteUserResponse | null,
    ) => void,
  ): UnaryResponse;
  listUsers(
    requestMessage: identity_pb.ListUsersRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListUsersResponse | null,
    ) => void,
  ): UnaryResponse;
  listUsers(
    requestMessage: identity_pb.ListUsersRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListUsersResponse | null,
    ) => void,
  ): UnaryResponse;
  updateUserPassword(
    requestMessage: identity_pb.UpdateUserPasswordRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateUserPasswordResponse | null,
    ) => void,
  ): UnaryResponse;
  updateUserPassword(
    requestMessage: identity_pb.UpdateUserPasswordRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateUserPasswordResponse | null,
    ) => void,
  ): UnaryResponse;
  createOrganization(
    requestMessage: identity_pb.CreateOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  createOrganization(
    requestMessage: identity_pb.CreateOrganizationRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  getOrganization(
    requestMessage: identity_pb.GetOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  getOrganization(
    requestMessage: identity_pb.GetOrganizationRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  updateOrganization(
    requestMessage: identity_pb.UpdateOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  updateOrganization(
    requestMessage: identity_pb.UpdateOrganizationRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteOrganization(
    requestMessage: identity_pb.DeleteOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteOrganization(
    requestMessage: identity_pb.DeleteOrganizationRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteOrganizationResponse | null,
    ) => void,
  ): UnaryResponse;
  listOrganizations(
    requestMessage: identity_pb.ListOrganizationsRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListOrganizationsResponse | null,
    ) => void,
  ): UnaryResponse;
  listOrganizations(
    requestMessage: identity_pb.ListOrganizationsRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListOrganizationsResponse | null,
    ) => void,
  ): UnaryResponse;
  addOrganizationUser(
    requestMessage: identity_pb.AddOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.AddOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  addOrganizationUser(
    requestMessage: identity_pb.AddOrganizationUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.AddOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  getOrganizationUser(
    requestMessage: identity_pb.GetOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  getOrganizationUser(
    requestMessage: identity_pb.GetOrganizationUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  updateOrganizationUser(
    requestMessage: identity_pb.UpdateOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  updateOrganizationUser(
    requestMessage: identity_pb.UpdateOrganizationUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.UpdateOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  removeOrganizationUser(
    requestMessage: identity_pb.RemoveOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.RemoveOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  removeOrganizationUser(
    requestMessage: identity_pb.RemoveOrganizationUserRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.RemoveOrganizationUserResponse | null,
    ) => void,
  ): UnaryResponse;
  listOrganizationUsers(
    requestMessage: identity_pb.ListOrganizationUsersRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListOrganizationUsersResponse | null,
    ) => void,
  ): UnaryResponse;
  listOrganizationUsers(
    requestMessage: identity_pb.ListOrganizationUsersRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListOrganizationUsersResponse | null,
    ) => void,
  ): UnaryResponse;
  listUserOrganizations(
    requestMessage: identity_pb.ListUserOrganizationsRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListUserOrganizationsResponse | null,
    ) => void,
  ): UnaryResponse;
  listUserOrganizations(
    requestMessage: identity_pb.ListUserOrganizationsRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListUserOrganizationsResponse | null,
    ) => void,
  ): UnaryResponse;
  createApiKey(
    requestMessage: identity_pb.CreateApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  createApiKey(
    requestMessage: identity_pb.CreateApiKeyRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.CreateApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  getApiKey(
    requestMessage: identity_pb.GetApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  getApiKey(
    requestMessage: identity_pb.GetApiKeyRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.GetApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteApiKey(
    requestMessage: identity_pb.DeleteApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  deleteApiKey(
    requestMessage: identity_pb.DeleteApiKeyRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.DeleteApiKeyResponse | null,
    ) => void,
  ): UnaryResponse;
  listApiKeys(
    requestMessage: identity_pb.ListApiKeysRequest,
    metadata: grpc.Metadata,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListApiKeysResponse | null,
    ) => void,
  ): UnaryResponse;
  listApiKeys(
    requestMessage: identity_pb.ListApiKeysRequest,
    callback: (
      error: ServiceError | null,
      responseMessage: identity_pb.ListApiKeysResponse | null,
    ) => void,
  ): UnaryResponse;
}
