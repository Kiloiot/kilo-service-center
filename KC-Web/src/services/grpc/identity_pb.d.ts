// package: kilocenter.api.v1
// file: identity.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_field_mask_pb from "google-protobuf/google/protobuf/field_mask_pb";

export class LoginRequest extends jspb.Message {
  getEmail(): string;
  setEmail(value: string): void;

  getPassword(): string;
  setPassword(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LoginRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LoginRequest,
  ): LoginRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: LoginRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LoginRequest;
  static deserializeBinaryFromReader(
    message: LoginRequest,
    reader: jspb.BinaryReader,
  ): LoginRequest;
}

export namespace LoginRequest {
  export type AsObject = {
    email: string;
    password: string;
  };
}

export class LoginResponse extends jspb.Message {
  hasTokens(): boolean;
  clearTokens(): void;
  getTokens(): AuthTokens | undefined;
  setTokens(value?: AuthTokens): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): UserProfile | undefined;
  setUser(value?: UserProfile): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LoginResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LoginResponse,
  ): LoginResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: LoginResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LoginResponse;
  static deserializeBinaryFromReader(
    message: LoginResponse,
    reader: jspb.BinaryReader,
  ): LoginResponse;
}

export namespace LoginResponse {
  export type AsObject = {
    tokens?: AuthTokens.AsObject;
    user?: UserProfile.AsObject;
  };
}

export class RefreshTokensRequest extends jspb.Message {
  getRefreshToken(): string;
  setRefreshToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RefreshTokensRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: RefreshTokensRequest,
  ): RefreshTokensRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: RefreshTokensRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): RefreshTokensRequest;
  static deserializeBinaryFromReader(
    message: RefreshTokensRequest,
    reader: jspb.BinaryReader,
  ): RefreshTokensRequest;
}

export namespace RefreshTokensRequest {
  export type AsObject = {
    refreshToken: string;
  };
}

export class RefreshTokensResponse extends jspb.Message {
  hasTokens(): boolean;
  clearTokens(): void;
  getTokens(): AuthTokens | undefined;
  setTokens(value?: AuthTokens): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RefreshTokensResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: RefreshTokensResponse,
  ): RefreshTokensResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: RefreshTokensResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): RefreshTokensResponse;
  static deserializeBinaryFromReader(
    message: RefreshTokensResponse,
    reader: jspb.BinaryReader,
  ): RefreshTokensResponse;
}

export namespace RefreshTokensResponse {
  export type AsObject = {
    tokens?: AuthTokens.AsObject;
  };
}

export class GetProfileRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetProfileRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetProfileRequest,
  ): GetProfileRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetProfileRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetProfileRequest;
  static deserializeBinaryFromReader(
    message: GetProfileRequest,
    reader: jspb.BinaryReader,
  ): GetProfileRequest;
}

export namespace GetProfileRequest {
  export type AsObject = {};
}

export class GetProfileResponse extends jspb.Message {
  hasUser(): boolean;
  clearUser(): void;
  getUser(): UserProfile | undefined;
  setUser(value?: UserProfile): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetProfileResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetProfileResponse,
  ): GetProfileResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetProfileResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetProfileResponse;
  static deserializeBinaryFromReader(
    message: GetProfileResponse,
    reader: jspb.BinaryReader,
  ): GetProfileResponse;
}

export namespace GetProfileResponse {
  export type AsObject = {
    user?: UserProfile.AsObject;
  };
}

export class GetAuthSettingsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAuthSettingsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetAuthSettingsRequest,
  ): GetAuthSettingsRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetAuthSettingsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetAuthSettingsRequest;
  static deserializeBinaryFromReader(
    message: GetAuthSettingsRequest,
    reader: jspb.BinaryReader,
  ): GetAuthSettingsRequest;
}

export namespace GetAuthSettingsRequest {
  export type AsObject = {};
}

export class GetAuthSettingsResponse extends jspb.Message {
  hasSettings(): boolean;
  clearSettings(): void;
  getSettings(): AuthSettings | undefined;
  setSettings(value?: AuthSettings): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAuthSettingsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetAuthSettingsResponse,
  ): GetAuthSettingsResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetAuthSettingsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetAuthSettingsResponse;
  static deserializeBinaryFromReader(
    message: GetAuthSettingsResponse,
    reader: jspb.BinaryReader,
  ): GetAuthSettingsResponse;
}

export namespace GetAuthSettingsResponse {
  export type AsObject = {
    settings?: AuthSettings.AsObject;
  };
}

export class LogoutRequest extends jspb.Message {
  getRefreshToken(): string;
  setRefreshToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogoutRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LogoutRequest,
  ): LogoutRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: LogoutRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LogoutRequest;
  static deserializeBinaryFromReader(
    message: LogoutRequest,
    reader: jspb.BinaryReader,
  ): LogoutRequest;
}

export namespace LogoutRequest {
  export type AsObject = {
    refreshToken: string;
  };
}

export class LogoutResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogoutResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LogoutResponse,
  ): LogoutResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: LogoutResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LogoutResponse;
  static deserializeBinaryFromReader(
    message: LogoutResponse,
    reader: jspb.BinaryReader,
  ): LogoutResponse;
}

export namespace LogoutResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ChangePasswordRequest extends jspb.Message {
  getCurrentPassword(): string;
  setCurrentPassword(value: string): void;

  getNewPassword(): string;
  setNewPassword(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChangePasswordRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ChangePasswordRequest,
  ): ChangePasswordRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ChangePasswordRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ChangePasswordRequest;
  static deserializeBinaryFromReader(
    message: ChangePasswordRequest,
    reader: jspb.BinaryReader,
  ): ChangePasswordRequest;
}

export namespace ChangePasswordRequest {
  export type AsObject = {
    currentPassword: string;
    newPassword: string;
  };
}

export class ChangePasswordResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChangePasswordResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ChangePasswordResponse,
  ): ChangePasswordResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ChangePasswordResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ChangePasswordResponse;
  static deserializeBinaryFromReader(
    message: ChangePasswordResponse,
    reader: jspb.BinaryReader,
  ): ChangePasswordResponse;
}

export namespace ChangePasswordResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ExchangeOIDCRequest extends jspb.Message {
  getCode(): string;
  setCode(value: string): void;

  getState(): string;
  setState(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExchangeOIDCRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ExchangeOIDCRequest,
  ): ExchangeOIDCRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ExchangeOIDCRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ExchangeOIDCRequest;
  static deserializeBinaryFromReader(
    message: ExchangeOIDCRequest,
    reader: jspb.BinaryReader,
  ): ExchangeOIDCRequest;
}

export namespace ExchangeOIDCRequest {
  export type AsObject = {
    code: string;
    state: string;
  };
}

export class ExchangeOAuth2Request extends jspb.Message {
  getCode(): string;
  setCode(value: string): void;

  getState(): string;
  setState(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExchangeOAuth2Request.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ExchangeOAuth2Request,
  ): ExchangeOAuth2Request.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ExchangeOAuth2Request,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ExchangeOAuth2Request;
  static deserializeBinaryFromReader(
    message: ExchangeOAuth2Request,
    reader: jspb.BinaryReader,
  ): ExchangeOAuth2Request;
}

export namespace ExchangeOAuth2Request {
  export type AsObject = {
    code: string;
    state: string;
  };
}

export class RegisterAccountRequest extends jspb.Message {
  getEmail(): string;
  setEmail(value: string): void;

  getPassword(): string;
  setPassword(value: string): void;

  getFirstName(): string;
  setFirstName(value: string): void;

  getLastName(): string;
  setLastName(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterAccountRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: RegisterAccountRequest,
  ): RegisterAccountRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: RegisterAccountRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): RegisterAccountRequest;
  static deserializeBinaryFromReader(
    message: RegisterAccountRequest,
    reader: jspb.BinaryReader,
  ): RegisterAccountRequest;
}

export namespace RegisterAccountRequest {
  export type AsObject = {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
    companyName: string;
  };
}

export class AuthTokens extends jspb.Message {
  getAccessToken(): string;
  setAccessToken(value: string): void;

  getRefreshToken(): string;
  setRefreshToken(value: string): void;

  getAccessExpiresIn(): number;
  setAccessExpiresIn(value: number): void;

  getRefreshExpiresIn(): number;
  setRefreshExpiresIn(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthTokens.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: AuthTokens,
  ): AuthTokens.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: AuthTokens,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): AuthTokens;
  static deserializeBinaryFromReader(
    message: AuthTokens,
    reader: jspb.BinaryReader,
  ): AuthTokens;
}

export namespace AuthTokens {
  export type AsObject = {
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
  };
}

export class UserProfile extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEmail(): string;
  setEmail(value: string): void;

  getIsAdmin(): boolean;
  setIsAdmin(value: boolean): void;

  getHasPassword(): boolean;
  setHasPassword(value: boolean): void;

  clearMembershipsList(): void;
  getMembershipsList(): Array<UserMembership>;
  setMembershipsList(value: Array<UserMembership>): void;
  addMemberships(value?: UserMembership, index?: number): UserMembership;

  getDefaultOrgId(): string;
  setDefaultOrgId(value: string): void;

  getFirstName(): string;
  setFirstName(value: string): void;

  getLastName(): string;
  setLastName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserProfile.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UserProfile,
  ): UserProfile.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UserProfile,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UserProfile;
  static deserializeBinaryFromReader(
    message: UserProfile,
    reader: jspb.BinaryReader,
  ): UserProfile;
}

export namespace UserProfile {
  export type AsObject = {
    id: string;
    email: string;
    isAdmin: boolean;
    hasPassword: boolean;
    membershipsList: Array<UserMembership.AsObject>;
    defaultOrgId: string;
    firstName: string;
    lastName: string;
  };
}

export class UserMembership extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getOrgName(): string;
  setOrgName(value: string): void;

  getRole(): string;
  setRole(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getIsOrgAdmin(): boolean;
  setIsOrgAdmin(value: boolean): void;

  getIsBaseStationAdmin(): boolean;
  setIsBaseStationAdmin(value: boolean): void;

  getIsEndpointAdmin(): boolean;
  setIsEndpointAdmin(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserMembership.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UserMembership,
  ): UserMembership.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UserMembership,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UserMembership;
  static deserializeBinaryFromReader(
    message: UserMembership,
    reader: jspb.BinaryReader,
  ): UserMembership;
}

export namespace UserMembership {
  export type AsObject = {
    orgId: string;
    orgName: string;
    role: string;
    status: string;
    isOrgAdmin: boolean;
    isBaseStationAdmin: boolean;
    isEndpointAdmin: boolean;
  };
}

export class AuthSettings extends jspb.Message {
  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  getLocalLoginEnabled(): boolean;
  setLocalLoginEnabled(value: boolean): void;

  getLoginUrl(): string;
  setLoginUrl(value: string): void;

  getLoginLabel(): string;
  setLoginLabel(value: string): void;

  getLoginRedirect(): boolean;
  setLoginRedirect(value: boolean): void;

  getLogoutUrl(): string;
  setLogoutUrl(value: string): void;

  getRefreshTokenEnabled(): boolean;
  setRefreshTokenEnabled(value: boolean): void;

  hasOidc(): boolean;
  clearOidc(): void;
  getOidc(): ProviderSettings | undefined;
  setOidc(value?: ProviderSettings): void;

  hasOauth2(): boolean;
  clearOauth2(): void;
  getOauth2(): ProviderSettings | undefined;
  setOauth2(value?: ProviderSettings): void;

  getRegistrationEnabled(): boolean;
  setRegistrationEnabled(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthSettings.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: AuthSettings,
  ): AuthSettings.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: AuthSettings,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): AuthSettings;
  static deserializeBinaryFromReader(
    message: AuthSettings,
    reader: jspb.BinaryReader,
  ): AuthSettings;
}

export namespace AuthSettings {
  export type AsObject = {
    enabled: boolean;
    localLoginEnabled: boolean;
    loginUrl: string;
    loginLabel: string;
    loginRedirect: boolean;
    logoutUrl: string;
    refreshTokenEnabled: boolean;
    oidc?: ProviderSettings.AsObject;
    oauth2?: ProviderSettings.AsObject;
    registrationEnabled: boolean;
  };
}

export class ProviderSettings extends jspb.Message {
  getEnabled(): boolean;
  setEnabled(value: boolean): void;

  getLoginUrl(): string;
  setLoginUrl(value: string): void;

  getLoginLabel(): string;
  setLoginLabel(value: string): void;

  getLoginRedirect(): boolean;
  setLoginRedirect(value: boolean): void;

  getLogoutUrl(): string;
  setLogoutUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProviderSettings.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ProviderSettings,
  ): ProviderSettings.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ProviderSettings,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ProviderSettings;
  static deserializeBinaryFromReader(
    message: ProviderSettings,
    reader: jspb.BinaryReader,
  ): ProviderSettings;
}

export namespace ProviderSettings {
  export type AsObject = {
    enabled: boolean;
    loginUrl: string;
    loginLabel: string;
    loginRedirect: boolean;
    logoutUrl: string;
  };
}

export class CreateUserRequest extends jspb.Message {
  getEmail(): string;
  setEmail(value: string): void;

  getPassword(): string;
  setPassword(value: string): void;

  getNote(): string;
  setNote(value: string): void;

  getIsAdmin(): boolean;
  setIsAdmin(value: boolean): void;

  getIsActive(): boolean;
  setIsActive(value: boolean): void;

  getIsTenantManager(): boolean;
  setIsTenantManager(value: boolean): void;

  getIsBaseStationManager(): boolean;
  setIsBaseStationManager(value: boolean): void;

  getIsEndpointManager(): boolean;
  setIsEndpointManager(value: boolean): void;

  getFirstName(): string;
  setFirstName(value: string): void;

  getLastName(): string;
  setLastName(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateUserRequest,
  ): CreateUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateUserRequest;
  static deserializeBinaryFromReader(
    message: CreateUserRequest,
    reader: jspb.BinaryReader,
  ): CreateUserRequest;
}

export namespace CreateUserRequest {
  export type AsObject = {
    email: string;
    password: string;
    note: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    firstName: string;
    lastName: string;
    companyName: string;
  };
}

export class CreateUserResponse extends jspb.Message {
  hasUser(): boolean;
  clearUser(): void;
  getUser(): User | undefined;
  setUser(value?: User): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateUserResponse,
  ): CreateUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateUserResponse;
  static deserializeBinaryFromReader(
    message: CreateUserResponse,
    reader: jspb.BinaryReader,
  ): CreateUserResponse;
}

export namespace CreateUserResponse {
  export type AsObject = {
    user?: User.AsObject;
  };
}

export class GetUserRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetUserRequest,
  ): GetUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetUserRequest;
  static deserializeBinaryFromReader(
    message: GetUserRequest,
    reader: jspb.BinaryReader,
  ): GetUserRequest;
}

export namespace GetUserRequest {
  export type AsObject = {
    id: string;
  };
}

export class GetUserResponse extends jspb.Message {
  hasUser(): boolean;
  clearUser(): void;
  getUser(): User | undefined;
  setUser(value?: User): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetUserResponse,
  ): GetUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetUserResponse;
  static deserializeBinaryFromReader(
    message: GetUserResponse,
    reader: jspb.BinaryReader,
  ): GetUserResponse;
}

export namespace GetUserResponse {
  export type AsObject = {
    user?: User.AsObject;
  };
}

export class UpdateUserRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEmail(): string;
  setEmail(value: string): void;

  getNote(): string;
  setNote(value: string): void;

  getIsAdmin(): boolean;
  setIsAdmin(value: boolean): void;

  getIsActive(): boolean;
  setIsActive(value: boolean): void;

  getIsTenantManager(): boolean;
  setIsTenantManager(value: boolean): void;

  getIsBaseStationManager(): boolean;
  setIsBaseStationManager(value: boolean): void;

  getIsEndpointManager(): boolean;
  setIsEndpointManager(value: boolean): void;

  hasUpdateMask(): boolean;
  clearUpdateMask(): void;
  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateUserRequest,
  ): UpdateUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateUserRequest;
  static deserializeBinaryFromReader(
    message: UpdateUserRequest,
    reader: jspb.BinaryReader,
  ): UpdateUserRequest;
}

export namespace UpdateUserRequest {
  export type AsObject = {
    id: string;
    email: string;
    note: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateUserResponse extends jspb.Message {
  hasUser(): boolean;
  clearUser(): void;
  getUser(): User | undefined;
  setUser(value?: User): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateUserResponse,
  ): UpdateUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateUserResponse;
  static deserializeBinaryFromReader(
    message: UpdateUserResponse,
    reader: jspb.BinaryReader,
  ): UpdateUserResponse;
}

export namespace UpdateUserResponse {
  export type AsObject = {
    user?: User.AsObject;
  };
}

export class DeleteUserRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteUserRequest,
  ): DeleteUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteUserRequest;
  static deserializeBinaryFromReader(
    message: DeleteUserRequest,
    reader: jspb.BinaryReader,
  ): DeleteUserRequest;
}

export namespace DeleteUserRequest {
  export type AsObject = {
    id: string;
  };
}

export class DeleteUserResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteUserResponse,
  ): DeleteUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteUserResponse;
  static deserializeBinaryFromReader(
    message: DeleteUserResponse,
    reader: jspb.BinaryReader,
  ): DeleteUserResponse;
}

export namespace DeleteUserResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ListUsersRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUsersRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListUsersRequest,
  ): ListUsersRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListUsersRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListUsersRequest;
  static deserializeBinaryFromReader(
    message: ListUsersRequest,
    reader: jspb.BinaryReader,
  ): ListUsersRequest;
}

export namespace ListUsersRequest {
  export type AsObject = {
    pageSize: number;
    pageToken: string;
  };
}

export class ListUsersResponse extends jspb.Message {
  clearUsersList(): void;
  getUsersList(): Array<User>;
  setUsersList(value: Array<User>): void;
  addUsers(value?: User, index?: number): User;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUsersResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListUsersResponse,
  ): ListUsersResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListUsersResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListUsersResponse;
  static deserializeBinaryFromReader(
    message: ListUsersResponse,
    reader: jspb.BinaryReader,
  ): ListUsersResponse;
}

export namespace ListUsersResponse {
  export type AsObject = {
    usersList: Array<User.AsObject>;
    nextPageToken: string;
    totalCount: number;
  };
}

export class UpdateUserPasswordRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getNewPassword(): string;
  setNewPassword(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateUserPasswordRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateUserPasswordRequest,
  ): UpdateUserPasswordRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateUserPasswordRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateUserPasswordRequest;
  static deserializeBinaryFromReader(
    message: UpdateUserPasswordRequest,
    reader: jspb.BinaryReader,
  ): UpdateUserPasswordRequest;
}

export namespace UpdateUserPasswordRequest {
  export type AsObject = {
    id: string;
    newPassword: string;
  };
}

export class UpdateUserPasswordResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateUserPasswordResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateUserPasswordResponse,
  ): UpdateUserPasswordResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateUserPasswordResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateUserPasswordResponse;
  static deserializeBinaryFromReader(
    message: UpdateUserPasswordResponse,
    reader: jspb.BinaryReader,
  ): UpdateUserPasswordResponse;
}

export namespace UpdateUserPasswordResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class User extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getEmail(): string;
  setEmail(value: string): void;

  getIsAdmin(): boolean;
  setIsAdmin(value: boolean): void;

  getIsActive(): boolean;
  setIsActive(value: boolean): void;

  getIsTenantManager(): boolean;
  setIsTenantManager(value: boolean): void;

  getIsBaseStationManager(): boolean;
  setIsBaseStationManager(value: boolean): void;

  getIsEndpointManager(): boolean;
  setIsEndpointManager(value: boolean): void;

  getNote(): string;
  setNote(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getFirstName(): string;
  setFirstName(value: string): void;

  getLastName(): string;
  setLastName(value: string): void;

  getCompanyName(): string;
  setCompanyName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): User.AsObject;
  static toObject(includeInstance: boolean, msg: User): User.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: User,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): User;
  static deserializeBinaryFromReader(
    message: User,
    reader: jspb.BinaryReader,
  ): User;
}

export namespace User {
  export type AsObject = {
    id: string;
    email: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    note: string;
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    firstName: string;
    lastName: string;
    companyName: string;
  };
}

export class CreateOrganizationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getCanHaveBaseStations(): boolean;
  setCanHaveBaseStations(value: boolean): void;

  getMaxBaseStationCount(): number;
  setMaxBaseStationCount(value: number): void;

  getMaxEndpointCount(): number;
  setMaxEndpointCount(value: number): void;

  getTagsMap(): jspb.Map<string, string>;
  clearTagsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateOrganizationRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateOrganizationRequest,
  ): CreateOrganizationRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateOrganizationRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateOrganizationRequest;
  static deserializeBinaryFromReader(
    message: CreateOrganizationRequest,
    reader: jspb.BinaryReader,
  ): CreateOrganizationRequest;
}

export namespace CreateOrganizationRequest {
  export type AsObject = {
    name: string;
    description: string;
    canHaveBaseStations: boolean;
    maxBaseStationCount: number;
    maxEndpointCount: number;
    tagsMap: Array<[string, string]>;
  };
}

export class CreateOrganizationResponse extends jspb.Message {
  hasOrganization(): boolean;
  clearOrganization(): void;
  getOrganization(): Organization | undefined;
  setOrganization(value?: Organization): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateOrganizationResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateOrganizationResponse,
  ): CreateOrganizationResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateOrganizationResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateOrganizationResponse;
  static deserializeBinaryFromReader(
    message: CreateOrganizationResponse,
    reader: jspb.BinaryReader,
  ): CreateOrganizationResponse;
}

export namespace CreateOrganizationResponse {
  export type AsObject = {
    organization?: Organization.AsObject;
  };
}

export class GetOrganizationRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOrganizationRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetOrganizationRequest,
  ): GetOrganizationRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetOrganizationRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetOrganizationRequest;
  static deserializeBinaryFromReader(
    message: GetOrganizationRequest,
    reader: jspb.BinaryReader,
  ): GetOrganizationRequest;
}

export namespace GetOrganizationRequest {
  export type AsObject = {
    id: string;
  };
}

export class GetOrganizationResponse extends jspb.Message {
  hasOrganization(): boolean;
  clearOrganization(): void;
  getOrganization(): Organization | undefined;
  setOrganization(value?: Organization): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOrganizationResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetOrganizationResponse,
  ): GetOrganizationResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetOrganizationResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetOrganizationResponse;
  static deserializeBinaryFromReader(
    message: GetOrganizationResponse,
    reader: jspb.BinaryReader,
  ): GetOrganizationResponse;
}

export namespace GetOrganizationResponse {
  export type AsObject = {
    organization?: Organization.AsObject;
  };
}

export class UpdateOrganizationRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getCanHaveBaseStations(): boolean;
  setCanHaveBaseStations(value: boolean): void;

  getMaxBaseStationCount(): number;
  setMaxBaseStationCount(value: number): void;

  getMaxEndpointCount(): number;
  setMaxEndpointCount(value: number): void;

  getTagsMap(): jspb.Map<string, string>;
  clearTagsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOrganizationRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateOrganizationRequest,
  ): UpdateOrganizationRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateOrganizationRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOrganizationRequest;
  static deserializeBinaryFromReader(
    message: UpdateOrganizationRequest,
    reader: jspb.BinaryReader,
  ): UpdateOrganizationRequest;
}

export namespace UpdateOrganizationRequest {
  export type AsObject = {
    id: string;
    name: string;
    description: string;
    canHaveBaseStations: boolean;
    maxBaseStationCount: number;
    maxEndpointCount: number;
    tagsMap: Array<[string, string]>;
  };
}

export class UpdateOrganizationResponse extends jspb.Message {
  hasOrganization(): boolean;
  clearOrganization(): void;
  getOrganization(): Organization | undefined;
  setOrganization(value?: Organization): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOrganizationResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateOrganizationResponse,
  ): UpdateOrganizationResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateOrganizationResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOrganizationResponse;
  static deserializeBinaryFromReader(
    message: UpdateOrganizationResponse,
    reader: jspb.BinaryReader,
  ): UpdateOrganizationResponse;
}

export namespace UpdateOrganizationResponse {
  export type AsObject = {
    organization?: Organization.AsObject;
  };
}

export class DeleteOrganizationRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteOrganizationRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteOrganizationRequest,
  ): DeleteOrganizationRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteOrganizationRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteOrganizationRequest;
  static deserializeBinaryFromReader(
    message: DeleteOrganizationRequest,
    reader: jspb.BinaryReader,
  ): DeleteOrganizationRequest;
}

export namespace DeleteOrganizationRequest {
  export type AsObject = {
    id: string;
  };
}

export class DeleteOrganizationResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteOrganizationResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteOrganizationResponse,
  ): DeleteOrganizationResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteOrganizationResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteOrganizationResponse;
  static deserializeBinaryFromReader(
    message: DeleteOrganizationResponse,
    reader: jspb.BinaryReader,
  ): DeleteOrganizationResponse;
}

export namespace DeleteOrganizationResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ListOrganizationsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getTenantId(): number;
  setTenantId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOrganizationsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOrganizationsRequest,
  ): ListOrganizationsRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListOrganizationsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListOrganizationsRequest;
  static deserializeBinaryFromReader(
    message: ListOrganizationsRequest,
    reader: jspb.BinaryReader,
  ): ListOrganizationsRequest;
}

export namespace ListOrganizationsRequest {
  export type AsObject = {
    pageSize: number;
    pageToken: string;
    tenantId: number;
  };
}

export class ListOrganizationsResponse extends jspb.Message {
  clearOrganizationsList(): void;
  getOrganizationsList(): Array<Organization>;
  setOrganizationsList(value: Array<Organization>): void;
  addOrganizations(value?: Organization, index?: number): Organization;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOrganizationsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOrganizationsResponse,
  ): ListOrganizationsResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListOrganizationsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListOrganizationsResponse;
  static deserializeBinaryFromReader(
    message: ListOrganizationsResponse,
    reader: jspb.BinaryReader,
  ): ListOrganizationsResponse;
}

export namespace ListOrganizationsResponse {
  export type AsObject = {
    organizationsList: Array<Organization.AsObject>;
    nextPageToken: string;
    totalCount: number;
  };
}

export class Organization extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getTenantId(): number;
  setTenantId(value: number): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getState(): string;
  setState(value: string): void;

  getExternalId(): string;
  setExternalId(value: string): void;

  getCanHaveBaseStations(): boolean;
  setCanHaveBaseStations(value: boolean): void;

  getMaxBaseStationCount(): number;
  setMaxBaseStationCount(value: number): void;

  getMaxEndpointCount(): number;
  setMaxEndpointCount(value: number): void;

  getTagsMap(): jspb.Map<string, string>;
  clearTagsMap(): void;
  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Organization.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: Organization,
  ): Organization.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: Organization,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): Organization;
  static deserializeBinaryFromReader(
    message: Organization,
    reader: jspb.BinaryReader,
  ): Organization;
}

export namespace Organization {
  export type AsObject = {
    id: string;
    tenantId: number;
    name: string;
    description: string;
    state: string;
    externalId: string;
    canHaveBaseStations: boolean;
    maxBaseStationCount: number;
    maxEndpointCount: number;
    tagsMap: Array<[string, string]>;
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class AddOrganizationUserRequest extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  getRole(): string;
  setRole(value: string): void;

  getIsOrgAdmin(): boolean;
  setIsOrgAdmin(value: boolean): void;

  getIsBaseStationAdmin(): boolean;
  setIsBaseStationAdmin(value: boolean): void;

  getIsEndpointAdmin(): boolean;
  setIsEndpointAdmin(value: boolean): void;

  getEmail(): string;
  setEmail(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddOrganizationUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: AddOrganizationUserRequest,
  ): AddOrganizationUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: AddOrganizationUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): AddOrganizationUserRequest;
  static deserializeBinaryFromReader(
    message: AddOrganizationUserRequest,
    reader: jspb.BinaryReader,
  ): AddOrganizationUserRequest;
}

export namespace AddOrganizationUserRequest {
  export type AsObject = {
    orgId: string;
    userId: string;
    role: string;
    isOrgAdmin: boolean;
    isBaseStationAdmin: boolean;
    isEndpointAdmin: boolean;
    email: string;
  };
}

export class AddOrganizationUserResponse extends jspb.Message {
  hasMember(): boolean;
  clearMember(): void;
  getMember(): OrganizationUser | undefined;
  setMember(value?: OrganizationUser): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddOrganizationUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: AddOrganizationUserResponse,
  ): AddOrganizationUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: AddOrganizationUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): AddOrganizationUserResponse;
  static deserializeBinaryFromReader(
    message: AddOrganizationUserResponse,
    reader: jspb.BinaryReader,
  ): AddOrganizationUserResponse;
}

export namespace AddOrganizationUserResponse {
  export type AsObject = {
    member?: OrganizationUser.AsObject;
  };
}

export class GetOrganizationUserRequest extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOrganizationUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetOrganizationUserRequest,
  ): GetOrganizationUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetOrganizationUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetOrganizationUserRequest;
  static deserializeBinaryFromReader(
    message: GetOrganizationUserRequest,
    reader: jspb.BinaryReader,
  ): GetOrganizationUserRequest;
}

export namespace GetOrganizationUserRequest {
  export type AsObject = {
    orgId: string;
    userId: string;
  };
}

export class GetOrganizationUserResponse extends jspb.Message {
  hasMember(): boolean;
  clearMember(): void;
  getMember(): OrganizationUser | undefined;
  setMember(value?: OrganizationUser): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOrganizationUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetOrganizationUserResponse,
  ): GetOrganizationUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetOrganizationUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetOrganizationUserResponse;
  static deserializeBinaryFromReader(
    message: GetOrganizationUserResponse,
    reader: jspb.BinaryReader,
  ): GetOrganizationUserResponse;
}

export namespace GetOrganizationUserResponse {
  export type AsObject = {
    member?: OrganizationUser.AsObject;
  };
}

export class UpdateOrganizationUserRequest extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  getRole(): string;
  setRole(value: string): void;

  getIsOrgAdmin(): boolean;
  setIsOrgAdmin(value: boolean): void;

  getIsBaseStationAdmin(): boolean;
  setIsBaseStationAdmin(value: boolean): void;

  getIsEndpointAdmin(): boolean;
  setIsEndpointAdmin(value: boolean): void;

  hasUpdateMask(): boolean;
  clearUpdateMask(): void;
  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOrganizationUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateOrganizationUserRequest,
  ): UpdateOrganizationUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateOrganizationUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOrganizationUserRequest;
  static deserializeBinaryFromReader(
    message: UpdateOrganizationUserRequest,
    reader: jspb.BinaryReader,
  ): UpdateOrganizationUserRequest;
}

export namespace UpdateOrganizationUserRequest {
  export type AsObject = {
    orgId: string;
    userId: string;
    role: string;
    isOrgAdmin: boolean;
    isBaseStationAdmin: boolean;
    isEndpointAdmin: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateOrganizationUserResponse extends jspb.Message {
  hasMember(): boolean;
  clearMember(): void;
  getMember(): OrganizationUser | undefined;
  setMember(value?: OrganizationUser): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOrganizationUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UpdateOrganizationUserResponse,
  ): UpdateOrganizationUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: UpdateOrganizationUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOrganizationUserResponse;
  static deserializeBinaryFromReader(
    message: UpdateOrganizationUserResponse,
    reader: jspb.BinaryReader,
  ): UpdateOrganizationUserResponse;
}

export namespace UpdateOrganizationUserResponse {
  export type AsObject = {
    member?: OrganizationUser.AsObject;
  };
}

export class RemoveOrganizationUserRequest extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveOrganizationUserRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: RemoveOrganizationUserRequest,
  ): RemoveOrganizationUserRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: RemoveOrganizationUserRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): RemoveOrganizationUserRequest;
  static deserializeBinaryFromReader(
    message: RemoveOrganizationUserRequest,
    reader: jspb.BinaryReader,
  ): RemoveOrganizationUserRequest;
}

export namespace RemoveOrganizationUserRequest {
  export type AsObject = {
    orgId: string;
    userId: string;
  };
}

export class RemoveOrganizationUserResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveOrganizationUserResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: RemoveOrganizationUserResponse,
  ): RemoveOrganizationUserResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: RemoveOrganizationUserResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): RemoveOrganizationUserResponse;
  static deserializeBinaryFromReader(
    message: RemoveOrganizationUserResponse,
    reader: jspb.BinaryReader,
  ): RemoveOrganizationUserResponse;
}

export namespace RemoveOrganizationUserResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ListOrganizationUsersRequest extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOrganizationUsersRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOrganizationUsersRequest,
  ): ListOrganizationUsersRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListOrganizationUsersRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListOrganizationUsersRequest;
  static deserializeBinaryFromReader(
    message: ListOrganizationUsersRequest,
    reader: jspb.BinaryReader,
  ): ListOrganizationUsersRequest;
}

export namespace ListOrganizationUsersRequest {
  export type AsObject = {
    orgId: string;
    status: string;
    pageSize: number;
    pageToken: string;
  };
}

export class ListOrganizationUsersResponse extends jspb.Message {
  clearMembersList(): void;
  getMembersList(): Array<OrganizationUser>;
  setMembersList(value: Array<OrganizationUser>): void;
  addMembers(value?: OrganizationUser, index?: number): OrganizationUser;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOrganizationUsersResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOrganizationUsersResponse,
  ): ListOrganizationUsersResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListOrganizationUsersResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListOrganizationUsersResponse;
  static deserializeBinaryFromReader(
    message: ListOrganizationUsersResponse,
    reader: jspb.BinaryReader,
  ): ListOrganizationUsersResponse;
}

export namespace ListOrganizationUsersResponse {
  export type AsObject = {
    membersList: Array<OrganizationUser.AsObject>;
    nextPageToken: string;
    totalCount: number;
  };
}

export class OrganizationUser extends jspb.Message {
  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  getEmail(): string;
  setEmail(value: string): void;

  getRole(): string;
  setRole(value: string): void;

  getStatus(): string;
  setStatus(value: string): void;

  getIsOrgAdmin(): boolean;
  setIsOrgAdmin(value: boolean): void;

  getIsBaseStationAdmin(): boolean;
  setIsBaseStationAdmin(value: boolean): void;

  getIsEndpointAdmin(): boolean;
  setIsEndpointAdmin(value: boolean): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasUpdatedAt(): boolean;
  clearUpdatedAt(): void;
  getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OrganizationUser.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: OrganizationUser,
  ): OrganizationUser.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: OrganizationUser,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): OrganizationUser;
  static deserializeBinaryFromReader(
    message: OrganizationUser,
    reader: jspb.BinaryReader,
  ): OrganizationUser;
}

export namespace OrganizationUser {
  export type AsObject = {
    orgId: string;
    userId: string;
    email: string;
    role: string;
    status: string;
    isOrgAdmin: boolean;
    isBaseStationAdmin: boolean;
    isEndpointAdmin: boolean;
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListUserOrganizationsRequest extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUserOrganizationsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListUserOrganizationsRequest,
  ): ListUserOrganizationsRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListUserOrganizationsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListUserOrganizationsRequest;
  static deserializeBinaryFromReader(
    message: ListUserOrganizationsRequest,
    reader: jspb.BinaryReader,
  ): ListUserOrganizationsRequest;
}

export namespace ListUserOrganizationsRequest {
  export type AsObject = {
    userId: string;
  };
}

export class ListUserOrganizationsResponse extends jspb.Message {
  clearMembershipsList(): void;
  getMembershipsList(): Array<UserMembership>;
  setMembershipsList(value: Array<UserMembership>): void;
  addMemberships(value?: UserMembership, index?: number): UserMembership;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUserOrganizationsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListUserOrganizationsResponse,
  ): ListUserOrganizationsResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListUserOrganizationsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListUserOrganizationsResponse;
  static deserializeBinaryFromReader(
    message: ListUserOrganizationsResponse,
    reader: jspb.BinaryReader,
  ): ListUserOrganizationsResponse;
}

export namespace ListUserOrganizationsResponse {
  export type AsObject = {
    membershipsList: Array<UserMembership.AsObject>;
  };
}

export class CreateApiKeyRequest extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getKeyType(): string;
  setKeyType(value: string): void;

  hasExpiresAt(): boolean;
  clearExpiresAt(): void;
  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateApiKeyRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateApiKeyRequest,
  ): CreateApiKeyRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateApiKeyRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateApiKeyRequest;
  static deserializeBinaryFromReader(
    message: CreateApiKeyRequest,
    reader: jspb.BinaryReader,
  ): CreateApiKeyRequest;
}

export namespace CreateApiKeyRequest {
  export type AsObject = {
    name: string;
    keyType: string;
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class CreateApiKeyResponse extends jspb.Message {
  hasApiKey(): boolean;
  clearApiKey(): void;
  getApiKey(): ApiKey | undefined;
  setApiKey(value?: ApiKey): void;

  getRawKey(): string;
  setRawKey(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateApiKeyResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: CreateApiKeyResponse,
  ): CreateApiKeyResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: CreateApiKeyResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): CreateApiKeyResponse;
  static deserializeBinaryFromReader(
    message: CreateApiKeyResponse,
    reader: jspb.BinaryReader,
  ): CreateApiKeyResponse;
}

export namespace CreateApiKeyResponse {
  export type AsObject = {
    apiKey?: ApiKey.AsObject;
    rawKey: string;
  };
}

export class GetApiKeyRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetApiKeyRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetApiKeyRequest,
  ): GetApiKeyRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetApiKeyRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetApiKeyRequest;
  static deserializeBinaryFromReader(
    message: GetApiKeyRequest,
    reader: jspb.BinaryReader,
  ): GetApiKeyRequest;
}

export namespace GetApiKeyRequest {
  export type AsObject = {
    id: string;
  };
}

export class GetApiKeyResponse extends jspb.Message {
  hasApiKey(): boolean;
  clearApiKey(): void;
  getApiKey(): ApiKey | undefined;
  setApiKey(value?: ApiKey): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetApiKeyResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetApiKeyResponse,
  ): GetApiKeyResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: GetApiKeyResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetApiKeyResponse;
  static deserializeBinaryFromReader(
    message: GetApiKeyResponse,
    reader: jspb.BinaryReader,
  ): GetApiKeyResponse;
}

export namespace GetApiKeyResponse {
  export type AsObject = {
    apiKey?: ApiKey.AsObject;
  };
}

export class DeleteApiKeyRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteApiKeyRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteApiKeyRequest,
  ): DeleteApiKeyRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteApiKeyRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteApiKeyRequest;
  static deserializeBinaryFromReader(
    message: DeleteApiKeyRequest,
    reader: jspb.BinaryReader,
  ): DeleteApiKeyRequest;
}

export namespace DeleteApiKeyRequest {
  export type AsObject = {
    id: string;
  };
}

export class DeleteApiKeyResponse extends jspb.Message {
  getSuccess(): boolean;
  setSuccess(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteApiKeyResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DeleteApiKeyResponse,
  ): DeleteApiKeyResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: DeleteApiKeyResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): DeleteApiKeyResponse;
  static deserializeBinaryFromReader(
    message: DeleteApiKeyResponse,
    reader: jspb.BinaryReader,
  ): DeleteApiKeyResponse;
}

export namespace DeleteApiKeyResponse {
  export type AsObject = {
    success: boolean;
  };
}

export class ListApiKeysRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): void;

  getPageToken(): string;
  setPageToken(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListApiKeysRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListApiKeysRequest,
  ): ListApiKeysRequest.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListApiKeysRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListApiKeysRequest;
  static deserializeBinaryFromReader(
    message: ListApiKeysRequest,
    reader: jspb.BinaryReader,
  ): ListApiKeysRequest;
}

export namespace ListApiKeysRequest {
  export type AsObject = {
    pageSize: number;
    pageToken: string;
    userId: string;
  };
}

export class ListApiKeysResponse extends jspb.Message {
  clearApiKeysList(): void;
  getApiKeysList(): Array<ApiKey>;
  setApiKeysList(value: Array<ApiKey>): void;
  addApiKeys(value?: ApiKey, index?: number): ApiKey;

  getNextPageToken(): string;
  setNextPageToken(value: string): void;

  getTotalCount(): number;
  setTotalCount(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListApiKeysResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListApiKeysResponse,
  ): ListApiKeysResponse.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ListApiKeysResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListApiKeysResponse;
  static deserializeBinaryFromReader(
    message: ListApiKeysResponse,
    reader: jspb.BinaryReader,
  ): ListApiKeysResponse;
}

export namespace ListApiKeysResponse {
  export type AsObject = {
    apiKeysList: Array<ApiKey.AsObject>;
    nextPageToken: string;
    totalCount: number;
  };
}

export class ApiKey extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getOrgId(): string;
  setOrgId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getKeyPrefix(): string;
  setKeyPrefix(value: string): void;

  getKeyType(): string;
  setKeyType(value: string): void;

  getIsActive(): boolean;
  setIsActive(value: boolean): void;

  hasExpiresAt(): boolean;
  clearExpiresAt(): void;
  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastUsedAt(): boolean;
  clearLastUsedAt(): void;
  getLastUsedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastUsedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ApiKey.AsObject;
  static toObject(includeInstance: boolean, msg: ApiKey): ApiKey.AsObject;
  static extensions: { [key: number]: jspb.ExtensionFieldInfo<jspb.Message> };
  static extensionsBinary: {
    [key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>;
  };
  static serializeBinaryToWriter(
    message: ApiKey,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ApiKey;
  static deserializeBinaryFromReader(
    message: ApiKey,
    reader: jspb.BinaryReader,
  ): ApiKey;
}

export namespace ApiKey {
  export type AsObject = {
    id: string;
    orgId: string;
    userId: string;
    name: string;
    keyPrefix: string;
    keyType: string;
    isActive: boolean;
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastUsedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}
