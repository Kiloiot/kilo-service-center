// source: identity.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = (function() {
  if (this) { return this; }
  if (typeof window !== 'undefined') { return window; }
  if (typeof global !== 'undefined') { return global; }
  if (typeof self !== 'undefined') { return self; }
  return Function('return this')();
}.call(null));

var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
goog.object.extend(proto, google_protobuf_timestamp_pb);
var google_protobuf_field_mask_pb = require('google-protobuf/google/protobuf/field_mask_pb.js');
goog.object.extend(proto, google_protobuf_field_mask_pb);
goog.exportSymbol('proto.kilocenter.api.v1.AddOrganizationUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.AddOrganizationUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ApiKey', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.AuthSettings', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.AuthTokens', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ChangePasswordRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ChangePasswordResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateApiKeyRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateApiKeyResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateOrganizationRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateOrganizationResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.CreateUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteApiKeyRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteApiKeyResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteOrganizationRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteOrganizationResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.DeleteUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ExchangeOAuth2Request', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ExchangeOIDCRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetApiKeyRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetApiKeyResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetAuthSettingsRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetAuthSettingsResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetOrganizationRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetOrganizationResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetOrganizationUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetOrganizationUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetProfileRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetProfileResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.GetUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListApiKeysRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListApiKeysResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListOrganizationUsersRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListOrganizationUsersResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListOrganizationsRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListOrganizationsResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListUserOrganizationsRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListUserOrganizationsResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListUsersRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ListUsersResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.LoginRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.LoginResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.LogoutRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.LogoutResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.Organization', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.OrganizationUser', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.ProviderSettings', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.RefreshTokensRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.RefreshTokensResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.RegisterAccountRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.RemoveOrganizationUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.RemoveOrganizationUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateOrganizationRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateOrganizationResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateOrganizationUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateOrganizationUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateUserPasswordRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateUserPasswordResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateUserRequest', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UpdateUserResponse', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.User', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UserMembership', null, global);
goog.exportSymbol('proto.kilocenter.api.v1.UserProfile', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.LoginRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.LoginRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.LoginRequest.displayName = 'proto.kilocenter.api.v1.LoginRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.LoginResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.LoginResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.LoginResponse.displayName = 'proto.kilocenter.api.v1.LoginResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.RefreshTokensRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.RefreshTokensRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.RefreshTokensRequest.displayName = 'proto.kilocenter.api.v1.RefreshTokensRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.RefreshTokensResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.RefreshTokensResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.RefreshTokensResponse.displayName = 'proto.kilocenter.api.v1.RefreshTokensResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetProfileRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetProfileRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetProfileRequest.displayName = 'proto.kilocenter.api.v1.GetProfileRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetProfileResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetProfileResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetProfileResponse.displayName = 'proto.kilocenter.api.v1.GetProfileResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetAuthSettingsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetAuthSettingsRequest.displayName = 'proto.kilocenter.api.v1.GetAuthSettingsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetAuthSettingsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetAuthSettingsResponse.displayName = 'proto.kilocenter.api.v1.GetAuthSettingsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.LogoutRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.LogoutRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.LogoutRequest.displayName = 'proto.kilocenter.api.v1.LogoutRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.LogoutResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.LogoutResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.LogoutResponse.displayName = 'proto.kilocenter.api.v1.LogoutResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ChangePasswordRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ChangePasswordRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ChangePasswordRequest.displayName = 'proto.kilocenter.api.v1.ChangePasswordRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ChangePasswordResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ChangePasswordResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ChangePasswordResponse.displayName = 'proto.kilocenter.api.v1.ChangePasswordResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ExchangeOIDCRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ExchangeOIDCRequest.displayName = 'proto.kilocenter.api.v1.ExchangeOIDCRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ExchangeOAuth2Request, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ExchangeOAuth2Request.displayName = 'proto.kilocenter.api.v1.ExchangeOAuth2Request';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.RegisterAccountRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.RegisterAccountRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.RegisterAccountRequest.displayName = 'proto.kilocenter.api.v1.RegisterAccountRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.AuthTokens = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.AuthTokens, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.AuthTokens.displayName = 'proto.kilocenter.api.v1.AuthTokens';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UserProfile = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.UserProfile.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.UserProfile, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UserProfile.displayName = 'proto.kilocenter.api.v1.UserProfile';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UserMembership = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UserMembership, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UserMembership.displayName = 'proto.kilocenter.api.v1.UserMembership';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.AuthSettings = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.AuthSettings, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.AuthSettings.displayName = 'proto.kilocenter.api.v1.AuthSettings';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ProviderSettings = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ProviderSettings, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ProviderSettings.displayName = 'proto.kilocenter.api.v1.ProviderSettings';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateUserRequest.displayName = 'proto.kilocenter.api.v1.CreateUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateUserResponse.displayName = 'proto.kilocenter.api.v1.CreateUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetUserRequest.displayName = 'proto.kilocenter.api.v1.GetUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetUserResponse.displayName = 'proto.kilocenter.api.v1.GetUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateUserRequest.displayName = 'proto.kilocenter.api.v1.UpdateUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateUserResponse.displayName = 'proto.kilocenter.api.v1.UpdateUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteUserRequest.displayName = 'proto.kilocenter.api.v1.DeleteUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteUserResponse.displayName = 'proto.kilocenter.api.v1.DeleteUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListUsersRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ListUsersRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListUsersRequest.displayName = 'proto.kilocenter.api.v1.ListUsersRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListUsersResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.ListUsersResponse.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.ListUsersResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListUsersResponse.displayName = 'proto.kilocenter.api.v1.ListUsersResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateUserPasswordRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateUserPasswordRequest.displayName = 'proto.kilocenter.api.v1.UpdateUserPasswordRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateUserPasswordResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateUserPasswordResponse.displayName = 'proto.kilocenter.api.v1.UpdateUserPasswordResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.User = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.User, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.User.displayName = 'proto.kilocenter.api.v1.User';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateOrganizationRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateOrganizationRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateOrganizationRequest.displayName = 'proto.kilocenter.api.v1.CreateOrganizationRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateOrganizationResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateOrganizationResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateOrganizationResponse.displayName = 'proto.kilocenter.api.v1.CreateOrganizationResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetOrganizationRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetOrganizationRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetOrganizationRequest.displayName = 'proto.kilocenter.api.v1.GetOrganizationRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetOrganizationResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetOrganizationResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetOrganizationResponse.displayName = 'proto.kilocenter.api.v1.GetOrganizationResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateOrganizationRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateOrganizationRequest.displayName = 'proto.kilocenter.api.v1.UpdateOrganizationRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateOrganizationResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateOrganizationResponse.displayName = 'proto.kilocenter.api.v1.UpdateOrganizationResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteOrganizationRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteOrganizationRequest.displayName = 'proto.kilocenter.api.v1.DeleteOrganizationRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteOrganizationResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteOrganizationResponse.displayName = 'proto.kilocenter.api.v1.DeleteOrganizationResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListOrganizationsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ListOrganizationsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListOrganizationsRequest.displayName = 'proto.kilocenter.api.v1.ListOrganizationsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListOrganizationsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.ListOrganizationsResponse.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.ListOrganizationsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListOrganizationsResponse.displayName = 'proto.kilocenter.api.v1.ListOrganizationsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.Organization = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.Organization, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.Organization.displayName = 'proto.kilocenter.api.v1.Organization';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.AddOrganizationUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.AddOrganizationUserRequest.displayName = 'proto.kilocenter.api.v1.AddOrganizationUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.AddOrganizationUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.AddOrganizationUserResponse.displayName = 'proto.kilocenter.api.v1.AddOrganizationUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetOrganizationUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetOrganizationUserRequest.displayName = 'proto.kilocenter.api.v1.GetOrganizationUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetOrganizationUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetOrganizationUserResponse.displayName = 'proto.kilocenter.api.v1.GetOrganizationUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateOrganizationUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateOrganizationUserRequest.displayName = 'proto.kilocenter.api.v1.UpdateOrganizationUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.UpdateOrganizationUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.UpdateOrganizationUserResponse.displayName = 'proto.kilocenter.api.v1.UpdateOrganizationUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.RemoveOrganizationUserRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.RemoveOrganizationUserRequest.displayName = 'proto.kilocenter.api.v1.RemoveOrganizationUserRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.RemoveOrganizationUserResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.RemoveOrganizationUserResponse.displayName = 'proto.kilocenter.api.v1.RemoveOrganizationUserResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ListOrganizationUsersRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListOrganizationUsersRequest.displayName = 'proto.kilocenter.api.v1.ListOrganizationUsersRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.ListOrganizationUsersResponse.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.ListOrganizationUsersResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListOrganizationUsersResponse.displayName = 'proto.kilocenter.api.v1.ListOrganizationUsersResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.OrganizationUser = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.OrganizationUser, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.OrganizationUser.displayName = 'proto.kilocenter.api.v1.OrganizationUser';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ListUserOrganizationsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListUserOrganizationsRequest.displayName = 'proto.kilocenter.api.v1.ListUserOrganizationsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.ListUserOrganizationsResponse.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.ListUserOrganizationsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListUserOrganizationsResponse.displayName = 'proto.kilocenter.api.v1.ListUserOrganizationsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateApiKeyRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateApiKeyRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateApiKeyRequest.displayName = 'proto.kilocenter.api.v1.CreateApiKeyRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.CreateApiKeyResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.CreateApiKeyResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.CreateApiKeyResponse.displayName = 'proto.kilocenter.api.v1.CreateApiKeyResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetApiKeyRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetApiKeyRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetApiKeyRequest.displayName = 'proto.kilocenter.api.v1.GetApiKeyRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.GetApiKeyResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.GetApiKeyResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.GetApiKeyResponse.displayName = 'proto.kilocenter.api.v1.GetApiKeyResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteApiKeyRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteApiKeyRequest.displayName = 'proto.kilocenter.api.v1.DeleteApiKeyRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.DeleteApiKeyResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.DeleteApiKeyResponse.displayName = 'proto.kilocenter.api.v1.DeleteApiKeyResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListApiKeysRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ListApiKeysRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListApiKeysRequest.displayName = 'proto.kilocenter.api.v1.ListApiKeysRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ListApiKeysResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.kilocenter.api.v1.ListApiKeysResponse.repeatedFields_, null);
};
goog.inherits(proto.kilocenter.api.v1.ListApiKeysResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ListApiKeysResponse.displayName = 'proto.kilocenter.api.v1.ListApiKeysResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.kilocenter.api.v1.ApiKey = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.kilocenter.api.v1.ApiKey, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.kilocenter.api.v1.ApiKey.displayName = 'proto.kilocenter.api.v1.ApiKey';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.LoginRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.LoginRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.LoginRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LoginRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    email: jspb.Message.getFieldWithDefault(msg, 1, ""),
    password: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.LoginRequest}
 */
proto.kilocenter.api.v1.LoginRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.LoginRequest;
  return proto.kilocenter.api.v1.LoginRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.LoginRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.LoginRequest}
 */
proto.kilocenter.api.v1.LoginRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPassword(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.LoginRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.LoginRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.LoginRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LoginRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPassword();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string email = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.LoginRequest.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.LoginRequest} returns this
 */
proto.kilocenter.api.v1.LoginRequest.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string password = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.LoginRequest.prototype.getPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.LoginRequest} returns this
 */
proto.kilocenter.api.v1.LoginRequest.prototype.setPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.LoginResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.LoginResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LoginResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    tokens: (f = msg.getTokens()) && proto.kilocenter.api.v1.AuthTokens.toObject(includeInstance, f),
    user: (f = msg.getUser()) && proto.kilocenter.api.v1.UserProfile.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.LoginResponse}
 */
proto.kilocenter.api.v1.LoginResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.LoginResponse;
  return proto.kilocenter.api.v1.LoginResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.LoginResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.LoginResponse}
 */
proto.kilocenter.api.v1.LoginResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.AuthTokens;
      reader.readMessage(value,proto.kilocenter.api.v1.AuthTokens.deserializeBinaryFromReader);
      msg.setTokens(value);
      break;
    case 2:
      var value = new proto.kilocenter.api.v1.UserProfile;
      reader.readMessage(value,proto.kilocenter.api.v1.UserProfile.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.LoginResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.LoginResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LoginResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTokens();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.AuthTokens.serializeBinaryToWriter
    );
  }
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.kilocenter.api.v1.UserProfile.serializeBinaryToWriter
    );
  }
};


/**
 * optional AuthTokens tokens = 1;
 * @return {?proto.kilocenter.api.v1.AuthTokens}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.getTokens = function() {
  return /** @type{?proto.kilocenter.api.v1.AuthTokens} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.AuthTokens, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.AuthTokens|undefined} value
 * @return {!proto.kilocenter.api.v1.LoginResponse} returns this
*/
proto.kilocenter.api.v1.LoginResponse.prototype.setTokens = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.LoginResponse} returns this
 */
proto.kilocenter.api.v1.LoginResponse.prototype.clearTokens = function() {
  return this.setTokens(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.hasTokens = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional UserProfile user = 2;
 * @return {?proto.kilocenter.api.v1.UserProfile}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.getUser = function() {
  return /** @type{?proto.kilocenter.api.v1.UserProfile} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.UserProfile, 2));
};


/**
 * @param {?proto.kilocenter.api.v1.UserProfile|undefined} value
 * @return {!proto.kilocenter.api.v1.LoginResponse} returns this
*/
proto.kilocenter.api.v1.LoginResponse.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.LoginResponse} returns this
 */
proto.kilocenter.api.v1.LoginResponse.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.LoginResponse.prototype.hasUser = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.RefreshTokensRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.RefreshTokensRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.RefreshTokensRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RefreshTokensRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    refreshToken: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.RefreshTokensRequest}
 */
proto.kilocenter.api.v1.RefreshTokensRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.RefreshTokensRequest;
  return proto.kilocenter.api.v1.RefreshTokensRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.RefreshTokensRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.RefreshTokensRequest}
 */
proto.kilocenter.api.v1.RefreshTokensRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRefreshToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.RefreshTokensRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.RefreshTokensRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.RefreshTokensRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RefreshTokensRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRefreshToken();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string refresh_token = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.RefreshTokensRequest.prototype.getRefreshToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RefreshTokensRequest} returns this
 */
proto.kilocenter.api.v1.RefreshTokensRequest.prototype.setRefreshToken = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.RefreshTokensResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.RefreshTokensResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RefreshTokensResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    tokens: (f = msg.getTokens()) && proto.kilocenter.api.v1.AuthTokens.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.RefreshTokensResponse}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.RefreshTokensResponse;
  return proto.kilocenter.api.v1.RefreshTokensResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.RefreshTokensResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.RefreshTokensResponse}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.AuthTokens;
      reader.readMessage(value,proto.kilocenter.api.v1.AuthTokens.deserializeBinaryFromReader);
      msg.setTokens(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.RefreshTokensResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.RefreshTokensResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RefreshTokensResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTokens();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.AuthTokens.serializeBinaryToWriter
    );
  }
};


/**
 * optional AuthTokens tokens = 1;
 * @return {?proto.kilocenter.api.v1.AuthTokens}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.getTokens = function() {
  return /** @type{?proto.kilocenter.api.v1.AuthTokens} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.AuthTokens, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.AuthTokens|undefined} value
 * @return {!proto.kilocenter.api.v1.RefreshTokensResponse} returns this
*/
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.setTokens = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.RefreshTokensResponse} returns this
 */
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.clearTokens = function() {
  return this.setTokens(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.RefreshTokensResponse.prototype.hasTokens = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetProfileRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetProfileRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetProfileRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetProfileRequest.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetProfileRequest}
 */
proto.kilocenter.api.v1.GetProfileRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetProfileRequest;
  return proto.kilocenter.api.v1.GetProfileRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetProfileRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetProfileRequest}
 */
proto.kilocenter.api.v1.GetProfileRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetProfileRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetProfileRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetProfileRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetProfileRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetProfileResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetProfileResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetProfileResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetProfileResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    user: (f = msg.getUser()) && proto.kilocenter.api.v1.UserProfile.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetProfileResponse}
 */
proto.kilocenter.api.v1.GetProfileResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetProfileResponse;
  return proto.kilocenter.api.v1.GetProfileResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetProfileResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetProfileResponse}
 */
proto.kilocenter.api.v1.GetProfileResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.UserProfile;
      reader.readMessage(value,proto.kilocenter.api.v1.UserProfile.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetProfileResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetProfileResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetProfileResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetProfileResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.UserProfile.serializeBinaryToWriter
    );
  }
};


/**
 * optional UserProfile user = 1;
 * @return {?proto.kilocenter.api.v1.UserProfile}
 */
proto.kilocenter.api.v1.GetProfileResponse.prototype.getUser = function() {
  return /** @type{?proto.kilocenter.api.v1.UserProfile} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.UserProfile, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.UserProfile|undefined} value
 * @return {!proto.kilocenter.api.v1.GetProfileResponse} returns this
*/
proto.kilocenter.api.v1.GetProfileResponse.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetProfileResponse} returns this
 */
proto.kilocenter.api.v1.GetProfileResponse.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetProfileResponse.prototype.hasUser = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetAuthSettingsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsRequest}
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetAuthSettingsRequest;
  return proto.kilocenter.api.v1.GetAuthSettingsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsRequest}
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetAuthSettingsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetAuthSettingsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetAuthSettingsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    settings: (f = msg.getSettings()) && proto.kilocenter.api.v1.AuthSettings.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsResponse}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetAuthSettingsResponse;
  return proto.kilocenter.api.v1.GetAuthSettingsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsResponse}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.AuthSettings;
      reader.readMessage(value,proto.kilocenter.api.v1.AuthSettings.deserializeBinaryFromReader);
      msg.setSettings(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetAuthSettingsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetAuthSettingsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSettings();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.AuthSettings.serializeBinaryToWriter
    );
  }
};


/**
 * optional AuthSettings settings = 1;
 * @return {?proto.kilocenter.api.v1.AuthSettings}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.getSettings = function() {
  return /** @type{?proto.kilocenter.api.v1.AuthSettings} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.AuthSettings, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.AuthSettings|undefined} value
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsResponse} returns this
*/
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.setSettings = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetAuthSettingsResponse} returns this
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.clearSettings = function() {
  return this.setSettings(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetAuthSettingsResponse.prototype.hasSettings = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.LogoutRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.LogoutRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.LogoutRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LogoutRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    refreshToken: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.LogoutRequest}
 */
proto.kilocenter.api.v1.LogoutRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.LogoutRequest;
  return proto.kilocenter.api.v1.LogoutRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.LogoutRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.LogoutRequest}
 */
proto.kilocenter.api.v1.LogoutRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRefreshToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.LogoutRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.LogoutRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.LogoutRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LogoutRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRefreshToken();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string refresh_token = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.LogoutRequest.prototype.getRefreshToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.LogoutRequest} returns this
 */
proto.kilocenter.api.v1.LogoutRequest.prototype.setRefreshToken = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.LogoutResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.LogoutResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.LogoutResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LogoutResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.LogoutResponse}
 */
proto.kilocenter.api.v1.LogoutResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.LogoutResponse;
  return proto.kilocenter.api.v1.LogoutResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.LogoutResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.LogoutResponse}
 */
proto.kilocenter.api.v1.LogoutResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.LogoutResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.LogoutResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.LogoutResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.LogoutResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.LogoutResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.LogoutResponse} returns this
 */
proto.kilocenter.api.v1.LogoutResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ChangePasswordRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ChangePasswordRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ChangePasswordRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    currentPassword: jspb.Message.getFieldWithDefault(msg, 1, ""),
    newPassword: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ChangePasswordRequest}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ChangePasswordRequest;
  return proto.kilocenter.api.v1.ChangePasswordRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ChangePasswordRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ChangePasswordRequest}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCurrentPassword(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNewPassword(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ChangePasswordRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ChangePasswordRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ChangePasswordRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCurrentPassword();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getNewPassword();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string current_password = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.getCurrentPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ChangePasswordRequest} returns this
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.setCurrentPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string new_password = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.getNewPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ChangePasswordRequest} returns this
 */
proto.kilocenter.api.v1.ChangePasswordRequest.prototype.setNewPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ChangePasswordResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ChangePasswordResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ChangePasswordResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ChangePasswordResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ChangePasswordResponse}
 */
proto.kilocenter.api.v1.ChangePasswordResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ChangePasswordResponse;
  return proto.kilocenter.api.v1.ChangePasswordResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ChangePasswordResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ChangePasswordResponse}
 */
proto.kilocenter.api.v1.ChangePasswordResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ChangePasswordResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ChangePasswordResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ChangePasswordResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ChangePasswordResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.ChangePasswordResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.ChangePasswordResponse} returns this
 */
proto.kilocenter.api.v1.ChangePasswordResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ExchangeOIDCRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ExchangeOIDCRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    code: jspb.Message.getFieldWithDefault(msg, 1, ""),
    state: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ExchangeOIDCRequest}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ExchangeOIDCRequest;
  return proto.kilocenter.api.v1.ExchangeOIDCRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ExchangeOIDCRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ExchangeOIDCRequest}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCode(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setState(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ExchangeOIDCRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ExchangeOIDCRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCode();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getState();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string code = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.getCode = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ExchangeOIDCRequest} returns this
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.setCode = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string state = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.getState = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ExchangeOIDCRequest} returns this
 */
proto.kilocenter.api.v1.ExchangeOIDCRequest.prototype.setState = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ExchangeOAuth2Request.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ExchangeOAuth2Request} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.toObject = function(includeInstance, msg) {
  var f, obj = {
    code: jspb.Message.getFieldWithDefault(msg, 1, ""),
    state: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ExchangeOAuth2Request}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ExchangeOAuth2Request;
  return proto.kilocenter.api.v1.ExchangeOAuth2Request.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ExchangeOAuth2Request} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ExchangeOAuth2Request}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCode(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setState(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ExchangeOAuth2Request.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ExchangeOAuth2Request} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCode();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getState();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string code = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.getCode = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ExchangeOAuth2Request} returns this
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.setCode = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string state = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.getState = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ExchangeOAuth2Request} returns this
 */
proto.kilocenter.api.v1.ExchangeOAuth2Request.prototype.setState = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.RegisterAccountRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.RegisterAccountRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RegisterAccountRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    email: jspb.Message.getFieldWithDefault(msg, 1, ""),
    password: jspb.Message.getFieldWithDefault(msg, 2, ""),
    firstName: jspb.Message.getFieldWithDefault(msg, 3, ""),
    lastName: jspb.Message.getFieldWithDefault(msg, 4, ""),
    companyName: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.RegisterAccountRequest;
  return proto.kilocenter.api.v1.RegisterAccountRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.RegisterAccountRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPassword(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setFirstName(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setLastName(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setCompanyName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.RegisterAccountRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.RegisterAccountRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RegisterAccountRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPassword();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getFirstName();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getLastName();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getCompanyName();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * optional string email = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest} returns this
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string password = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.getPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest} returns this
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.setPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string first_name = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.getFirstName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest} returns this
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.setFirstName = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string last_name = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.getLastName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest} returns this
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.setLastName = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string company_name = 5;
 * @return {string}
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.getCompanyName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RegisterAccountRequest} returns this
 */
proto.kilocenter.api.v1.RegisterAccountRequest.prototype.setCompanyName = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.AuthTokens.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.AuthTokens} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AuthTokens.toObject = function(includeInstance, msg) {
  var f, obj = {
    accessToken: jspb.Message.getFieldWithDefault(msg, 1, ""),
    refreshToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    accessExpiresIn: jspb.Message.getFieldWithDefault(msg, 3, 0),
    refreshExpiresIn: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.AuthTokens}
 */
proto.kilocenter.api.v1.AuthTokens.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.AuthTokens;
  return proto.kilocenter.api.v1.AuthTokens.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.AuthTokens} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.AuthTokens}
 */
proto.kilocenter.api.v1.AuthTokens.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setAccessToken(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRefreshToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setAccessExpiresIn(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setRefreshExpiresIn(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.AuthTokens.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.AuthTokens} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AuthTokens.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getAccessToken();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRefreshToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getAccessExpiresIn();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
  f = message.getRefreshExpiresIn();
  if (f !== 0) {
    writer.writeInt64(
      4,
      f
    );
  }
};


/**
 * optional string access_token = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.getAccessToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AuthTokens} returns this
 */
proto.kilocenter.api.v1.AuthTokens.prototype.setAccessToken = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string refresh_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.getRefreshToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AuthTokens} returns this
 */
proto.kilocenter.api.v1.AuthTokens.prototype.setRefreshToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int64 access_expires_in = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.getAccessExpiresIn = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.AuthTokens} returns this
 */
proto.kilocenter.api.v1.AuthTokens.prototype.setAccessExpiresIn = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int64 refresh_expires_in = 4;
 * @return {number}
 */
proto.kilocenter.api.v1.AuthTokens.prototype.getRefreshExpiresIn = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.AuthTokens} returns this
 */
proto.kilocenter.api.v1.AuthTokens.prototype.setRefreshExpiresIn = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.UserProfile.repeatedFields_ = [5];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UserProfile.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UserProfile.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UserProfile} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UserProfile.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    email: jspb.Message.getFieldWithDefault(msg, 2, ""),
    isAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    hasPassword: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    membershipsList: jspb.Message.toObjectList(msg.getMembershipsList(),
    proto.kilocenter.api.v1.UserMembership.toObject, includeInstance),
    defaultOrgId: jspb.Message.getFieldWithDefault(msg, 6, ""),
    firstName: jspb.Message.getFieldWithDefault(msg, 7, ""),
    lastName: jspb.Message.getFieldWithDefault(msg, 8, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UserProfile}
 */
proto.kilocenter.api.v1.UserProfile.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UserProfile;
  return proto.kilocenter.api.v1.UserProfile.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UserProfile} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UserProfile}
 */
proto.kilocenter.api.v1.UserProfile.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAdmin(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setHasPassword(value);
      break;
    case 5:
      var value = new proto.kilocenter.api.v1.UserMembership;
      reader.readMessage(value,proto.kilocenter.api.v1.UserMembership.deserializeBinaryFromReader);
      msg.addMemberships(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setDefaultOrgId(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setFirstName(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setLastName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UserProfile.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UserProfile.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UserProfile} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UserProfile.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getIsAdmin();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getHasPassword();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getMembershipsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      5,
      f,
      proto.kilocenter.api.v1.UserMembership.serializeBinaryToWriter
    );
  }
  f = message.getDefaultOrgId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getFirstName();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getLastName();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string email = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool is_admin = 3;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getIsAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setIsAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool has_password = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getHasPassword = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setHasPassword = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * repeated UserMembership memberships = 5;
 * @return {!Array<!proto.kilocenter.api.v1.UserMembership>}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getMembershipsList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.UserMembership>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.UserMembership, 5));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.UserMembership>} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
*/
proto.kilocenter.api.v1.UserProfile.prototype.setMembershipsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 5, value);
};


/**
 * @param {!proto.kilocenter.api.v1.UserMembership=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.UserMembership}
 */
proto.kilocenter.api.v1.UserProfile.prototype.addMemberships = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 5, opt_value, proto.kilocenter.api.v1.UserMembership, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.clearMembershipsList = function() {
  return this.setMembershipsList([]);
};


/**
 * optional string default_org_id = 6;
 * @return {string}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getDefaultOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setDefaultOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string first_name = 7;
 * @return {string}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getFirstName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setFirstName = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional string last_name = 8;
 * @return {string}
 */
proto.kilocenter.api.v1.UserProfile.prototype.getLastName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserProfile} returns this
 */
proto.kilocenter.api.v1.UserProfile.prototype.setLastName = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UserMembership.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UserMembership.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UserMembership} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UserMembership.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    orgName: jspb.Message.getFieldWithDefault(msg, 2, ""),
    role: jspb.Message.getFieldWithDefault(msg, 3, ""),
    status: jspb.Message.getFieldWithDefault(msg, 4, ""),
    isOrgAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isBaseStationAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isEndpointAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 7, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UserMembership}
 */
proto.kilocenter.api.v1.UserMembership.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UserMembership;
  return proto.kilocenter.api.v1.UserMembership.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UserMembership} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UserMembership}
 */
proto.kilocenter.api.v1.UserMembership.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setRole(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setStatus(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsOrgAdmin(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationAdmin(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointAdmin(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UserMembership.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UserMembership.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UserMembership} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UserMembership.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOrgName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRole();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getStatus();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getIsOrgAdmin();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsBaseStationAdmin();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsEndpointAdmin();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string org_name = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getOrgName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setOrgName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string role = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getRole = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setRole = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string status = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getStatus = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setStatus = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional bool is_org_admin = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getIsOrgAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setIsOrgAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_base_station_admin = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getIsBaseStationAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setIsBaseStationAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_endpoint_admin = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UserMembership.prototype.getIsEndpointAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UserMembership} returns this
 */
proto.kilocenter.api.v1.UserMembership.prototype.setIsEndpointAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.AuthSettings.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.AuthSettings} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AuthSettings.toObject = function(includeInstance, msg) {
  var f, obj = {
    enabled: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    localLoginEnabled: jspb.Message.getBooleanFieldWithDefault(msg, 2, false),
    loginUrl: jspb.Message.getFieldWithDefault(msg, 3, ""),
    loginLabel: jspb.Message.getFieldWithDefault(msg, 4, ""),
    loginRedirect: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    logoutUrl: jspb.Message.getFieldWithDefault(msg, 6, ""),
    refreshTokenEnabled: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    oidc: (f = msg.getOidc()) && proto.kilocenter.api.v1.ProviderSettings.toObject(includeInstance, f),
    oauth2: (f = msg.getOauth2()) && proto.kilocenter.api.v1.ProviderSettings.toObject(includeInstance, f),
    registrationEnabled: jspb.Message.getBooleanFieldWithDefault(msg, 10, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.AuthSettings}
 */
proto.kilocenter.api.v1.AuthSettings.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.AuthSettings;
  return proto.kilocenter.api.v1.AuthSettings.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.AuthSettings} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.AuthSettings}
 */
proto.kilocenter.api.v1.AuthSettings.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setEnabled(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setLocalLoginEnabled(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setLoginUrl(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setLoginLabel(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setLoginRedirect(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setLogoutUrl(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setRefreshTokenEnabled(value);
      break;
    case 8:
      var value = new proto.kilocenter.api.v1.ProviderSettings;
      reader.readMessage(value,proto.kilocenter.api.v1.ProviderSettings.deserializeBinaryFromReader);
      msg.setOidc(value);
      break;
    case 9:
      var value = new proto.kilocenter.api.v1.ProviderSettings;
      reader.readMessage(value,proto.kilocenter.api.v1.ProviderSettings.deserializeBinaryFromReader);
      msg.setOauth2(value);
      break;
    case 10:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setRegistrationEnabled(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.AuthSettings.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.AuthSettings} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AuthSettings.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEnabled();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getLocalLoginEnabled();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getLoginUrl();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getLoginLabel();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getLoginRedirect();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getLogoutUrl();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getRefreshTokenEnabled();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getOidc();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      proto.kilocenter.api.v1.ProviderSettings.serializeBinaryToWriter
    );
  }
  f = message.getOauth2();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      proto.kilocenter.api.v1.ProviderSettings.serializeBinaryToWriter
    );
  }
  f = message.getRegistrationEnabled();
  if (f) {
    writer.writeBool(
      10,
      f
    );
  }
};


/**
 * optional bool enabled = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional bool local_login_enabled = 2;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getLocalLoginEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setLocalLoginEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};


/**
 * optional string login_url = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getLoginUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setLoginUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string login_label = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getLoginLabel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setLoginLabel = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional bool login_redirect = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getLoginRedirect = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setLoginRedirect = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional string logout_url = 6;
 * @return {string}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getLogoutUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setLogoutUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional bool refresh_token_enabled = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getRefreshTokenEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setRefreshTokenEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional ProviderSettings oidc = 8;
 * @return {?proto.kilocenter.api.v1.ProviderSettings}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getOidc = function() {
  return /** @type{?proto.kilocenter.api.v1.ProviderSettings} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.ProviderSettings, 8));
};


/**
 * @param {?proto.kilocenter.api.v1.ProviderSettings|undefined} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
*/
proto.kilocenter.api.v1.AuthSettings.prototype.setOidc = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.clearOidc = function() {
  return this.setOidc(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.hasOidc = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional ProviderSettings oauth2 = 9;
 * @return {?proto.kilocenter.api.v1.ProviderSettings}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getOauth2 = function() {
  return /** @type{?proto.kilocenter.api.v1.ProviderSettings} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.ProviderSettings, 9));
};


/**
 * @param {?proto.kilocenter.api.v1.ProviderSettings|undefined} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
*/
proto.kilocenter.api.v1.AuthSettings.prototype.setOauth2 = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.clearOauth2 = function() {
  return this.setOauth2(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.hasOauth2 = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional bool registration_enabled = 10;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AuthSettings.prototype.getRegistrationEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 10, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AuthSettings} returns this
 */
proto.kilocenter.api.v1.AuthSettings.prototype.setRegistrationEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 10, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ProviderSettings.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ProviderSettings} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ProviderSettings.toObject = function(includeInstance, msg) {
  var f, obj = {
    enabled: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    loginUrl: jspb.Message.getFieldWithDefault(msg, 2, ""),
    loginLabel: jspb.Message.getFieldWithDefault(msg, 3, ""),
    loginRedirect: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    logoutUrl: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ProviderSettings}
 */
proto.kilocenter.api.v1.ProviderSettings.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ProviderSettings;
  return proto.kilocenter.api.v1.ProviderSettings.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ProviderSettings} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ProviderSettings}
 */
proto.kilocenter.api.v1.ProviderSettings.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setEnabled(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setLoginUrl(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setLoginLabel(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setLoginRedirect(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setLogoutUrl(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ProviderSettings.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ProviderSettings} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ProviderSettings.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEnabled();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getLoginUrl();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getLoginLabel();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getLoginRedirect();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getLogoutUrl();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * optional bool enabled = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.getEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.ProviderSettings} returns this
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.setEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string login_url = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.getLoginUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ProviderSettings} returns this
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.setLoginUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string login_label = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.getLoginLabel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ProviderSettings} returns this
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.setLoginLabel = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool login_redirect = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.getLoginRedirect = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.ProviderSettings} returns this
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.setLoginRedirect = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional string logout_url = 5;
 * @return {string}
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.getLogoutUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ProviderSettings} returns this
 */
proto.kilocenter.api.v1.ProviderSettings.prototype.setLogoutUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    email: jspb.Message.getFieldWithDefault(msg, 1, ""),
    password: jspb.Message.getFieldWithDefault(msg, 2, ""),
    note: jspb.Message.getFieldWithDefault(msg, 3, ""),
    isAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isActive: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isTenantManager: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isBaseStationManager: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    isEndpointManager: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    firstName: jspb.Message.getFieldWithDefault(msg, 9, ""),
    lastName: jspb.Message.getFieldWithDefault(msg, 10, ""),
    companyName: jspb.Message.getFieldWithDefault(msg, 11, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateUserRequest}
 */
proto.kilocenter.api.v1.CreateUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateUserRequest;
  return proto.kilocenter.api.v1.CreateUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateUserRequest}
 */
proto.kilocenter.api.v1.CreateUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPassword(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setNote(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAdmin(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsActive(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsTenantManager(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationManager(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointManager(value);
      break;
    case 9:
      var value = /** @type {string} */ (reader.readString());
      msg.setFirstName(value);
      break;
    case 10:
      var value = /** @type {string} */ (reader.readString());
      msg.setLastName(value);
      break;
    case 11:
      var value = /** @type {string} */ (reader.readString());
      msg.setCompanyName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPassword();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getNote();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getIsAdmin();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsActive();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsTenantManager();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsBaseStationManager();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getIsEndpointManager();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getFirstName();
  if (f.length > 0) {
    writer.writeString(
      9,
      f
    );
  }
  f = message.getLastName();
  if (f.length > 0) {
    writer.writeString(
      10,
      f
    );
  }
  f = message.getCompanyName();
  if (f.length > 0) {
    writer.writeString(
      11,
      f
    );
  }
};


/**
 * optional string email = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string password = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string note = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getNote = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setNote = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool is_admin = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getIsAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setIsAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_active = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getIsActive = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setIsActive = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_tenant_manager = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getIsTenantManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setIsTenantManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_base_station_manager = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getIsBaseStationManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setIsBaseStationManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional bool is_endpoint_manager = 8;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getIsEndpointManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setIsEndpointManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional string first_name = 9;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getFirstName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 9, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setFirstName = function(value) {
  return jspb.Message.setProto3StringField(this, 9, value);
};


/**
 * optional string last_name = 10;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getLastName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 10, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setLastName = function(value) {
  return jspb.Message.setProto3StringField(this, 10, value);
};


/**
 * optional string company_name = 11;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.getCompanyName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 11, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateUserRequest} returns this
 */
proto.kilocenter.api.v1.CreateUserRequest.prototype.setCompanyName = function(value) {
  return jspb.Message.setProto3StringField(this, 11, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    user: (f = msg.getUser()) && proto.kilocenter.api.v1.User.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateUserResponse}
 */
proto.kilocenter.api.v1.CreateUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateUserResponse;
  return proto.kilocenter.api.v1.CreateUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateUserResponse}
 */
proto.kilocenter.api.v1.CreateUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.User;
      reader.readMessage(value,proto.kilocenter.api.v1.User.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.User.serializeBinaryToWriter
    );
  }
};


/**
 * optional User user = 1;
 * @return {?proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.CreateUserResponse.prototype.getUser = function() {
  return /** @type{?proto.kilocenter.api.v1.User} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.User, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.User|undefined} value
 * @return {!proto.kilocenter.api.v1.CreateUserResponse} returns this
*/
proto.kilocenter.api.v1.CreateUserResponse.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.CreateUserResponse} returns this
 */
proto.kilocenter.api.v1.CreateUserResponse.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateUserResponse.prototype.hasUser = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetUserRequest}
 */
proto.kilocenter.api.v1.GetUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetUserRequest;
  return proto.kilocenter.api.v1.GetUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetUserRequest}
 */
proto.kilocenter.api.v1.GetUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.GetUserRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.GetUserRequest} returns this
 */
proto.kilocenter.api.v1.GetUserRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    user: (f = msg.getUser()) && proto.kilocenter.api.v1.User.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetUserResponse}
 */
proto.kilocenter.api.v1.GetUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetUserResponse;
  return proto.kilocenter.api.v1.GetUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetUserResponse}
 */
proto.kilocenter.api.v1.GetUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.User;
      reader.readMessage(value,proto.kilocenter.api.v1.User.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.User.serializeBinaryToWriter
    );
  }
};


/**
 * optional User user = 1;
 * @return {?proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.GetUserResponse.prototype.getUser = function() {
  return /** @type{?proto.kilocenter.api.v1.User} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.User, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.User|undefined} value
 * @return {!proto.kilocenter.api.v1.GetUserResponse} returns this
*/
proto.kilocenter.api.v1.GetUserResponse.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetUserResponse} returns this
 */
proto.kilocenter.api.v1.GetUserResponse.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetUserResponse.prototype.hasUser = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    email: jspb.Message.getFieldWithDefault(msg, 2, ""),
    note: jspb.Message.getFieldWithDefault(msg, 3, ""),
    isAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isActive: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isTenantManager: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isBaseStationManager: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    isEndpointManager: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    updateMask: (f = msg.getUpdateMask()) && google_protobuf_field_mask_pb.FieldMask.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest}
 */
proto.kilocenter.api.v1.UpdateUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateUserRequest;
  return proto.kilocenter.api.v1.UpdateUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest}
 */
proto.kilocenter.api.v1.UpdateUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setNote(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAdmin(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsActive(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsTenantManager(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationManager(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointManager(value);
      break;
    case 9:
      var value = new google_protobuf_field_mask_pb.FieldMask;
      reader.readMessage(value,google_protobuf_field_mask_pb.FieldMask.deserializeBinaryFromReader);
      msg.setUpdateMask(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getNote();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getIsAdmin();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsActive();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsTenantManager();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsBaseStationManager();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getIsEndpointManager();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getUpdateMask();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_field_mask_pb.FieldMask.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string email = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string note = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getNote = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setNote = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool is_admin = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getIsAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setIsAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_active = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getIsActive = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setIsActive = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_tenant_manager = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getIsTenantManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setIsTenantManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_base_station_manager = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getIsBaseStationManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setIsBaseStationManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional bool is_endpoint_manager = 8;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getIsEndpointManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setIsEndpointManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional google.protobuf.FieldMask update_mask = 9;
 * @return {?proto.google.protobuf.FieldMask}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.getUpdateMask = function() {
  return /** @type{?proto.google.protobuf.FieldMask} */ (
    jspb.Message.getWrapperField(this, google_protobuf_field_mask_pb.FieldMask, 9));
};


/**
 * @param {?proto.google.protobuf.FieldMask|undefined} value
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
*/
proto.kilocenter.api.v1.UpdateUserRequest.prototype.setUpdateMask = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.UpdateUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.clearUpdateMask = function() {
  return this.setUpdateMask(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserRequest.prototype.hasUpdateMask = function() {
  return jspb.Message.getField(this, 9) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    user: (f = msg.getUser()) && proto.kilocenter.api.v1.User.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateUserResponse}
 */
proto.kilocenter.api.v1.UpdateUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateUserResponse;
  return proto.kilocenter.api.v1.UpdateUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateUserResponse}
 */
proto.kilocenter.api.v1.UpdateUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.User;
      reader.readMessage(value,proto.kilocenter.api.v1.User.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.User.serializeBinaryToWriter
    );
  }
};


/**
 * optional User user = 1;
 * @return {?proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.UpdateUserResponse.prototype.getUser = function() {
  return /** @type{?proto.kilocenter.api.v1.User} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.User, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.User|undefined} value
 * @return {!proto.kilocenter.api.v1.UpdateUserResponse} returns this
*/
proto.kilocenter.api.v1.UpdateUserResponse.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.UpdateUserResponse} returns this
 */
proto.kilocenter.api.v1.UpdateUserResponse.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserResponse.prototype.hasUser = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteUserRequest}
 */
proto.kilocenter.api.v1.DeleteUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteUserRequest;
  return proto.kilocenter.api.v1.DeleteUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteUserRequest}
 */
proto.kilocenter.api.v1.DeleteUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.DeleteUserRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.DeleteUserRequest} returns this
 */
proto.kilocenter.api.v1.DeleteUserRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteUserResponse}
 */
proto.kilocenter.api.v1.DeleteUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteUserResponse;
  return proto.kilocenter.api.v1.DeleteUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteUserResponse}
 */
proto.kilocenter.api.v1.DeleteUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.DeleteUserResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.DeleteUserResponse} returns this
 */
proto.kilocenter.api.v1.DeleteUserResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListUsersRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListUsersRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUsersRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    pageSize: jspb.Message.getFieldWithDefault(msg, 1, 0),
    pageToken: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListUsersRequest}
 */
proto.kilocenter.api.v1.ListUsersRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListUsersRequest;
  return proto.kilocenter.api.v1.ListUsersRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListUsersRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListUsersRequest}
 */
proto.kilocenter.api.v1.ListUsersRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setPageSize(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListUsersRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListUsersRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUsersRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPageSize();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional int32 page_size = 1;
 * @return {number}
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.getPageSize = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.setPageSize = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.getPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListUsersRequest.prototype.setPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.ListUsersResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListUsersResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListUsersResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUsersResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    usersList: jspb.Message.toObjectList(msg.getUsersList(),
    proto.kilocenter.api.v1.User.toObject, includeInstance),
    nextPageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    totalCount: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListUsersResponse}
 */
proto.kilocenter.api.v1.ListUsersResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListUsersResponse;
  return proto.kilocenter.api.v1.ListUsersResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListUsersResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListUsersResponse}
 */
proto.kilocenter.api.v1.ListUsersResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.User;
      reader.readMessage(value,proto.kilocenter.api.v1.User.deserializeBinaryFromReader);
      msg.addUsers(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTotalCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListUsersResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListUsersResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUsersResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsersList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.kilocenter.api.v1.User.serializeBinaryToWriter
    );
  }
  f = message.getNextPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTotalCount();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
};


/**
 * repeated User users = 1;
 * @return {!Array<!proto.kilocenter.api.v1.User>}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.getUsersList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.User>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.User, 1));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.User>} value
 * @return {!proto.kilocenter.api.v1.ListUsersResponse} returns this
*/
proto.kilocenter.api.v1.ListUsersResponse.prototype.setUsersList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.kilocenter.api.v1.User=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.addUsers = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.kilocenter.api.v1.User, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.ListUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.clearUsersList = function() {
  return this.setUsersList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 total_count = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.getTotalCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListUsersResponse.prototype.setTotalCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateUserPasswordRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    newPassword: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordRequest}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateUserPasswordRequest;
  return proto.kilocenter.api.v1.UpdateUserPasswordRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordRequest}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNewPassword(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateUserPasswordRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getNewPassword();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string new_password = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.getNewPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordRequest} returns this
 */
proto.kilocenter.api.v1.UpdateUserPasswordRequest.prototype.setNewPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateUserPasswordResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordResponse}
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateUserPasswordResponse;
  return proto.kilocenter.api.v1.UpdateUserPasswordResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordResponse}
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateUserPasswordResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateUserPasswordResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateUserPasswordResponse} returns this
 */
proto.kilocenter.api.v1.UpdateUserPasswordResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.User.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.User.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.User} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.User.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    email: jspb.Message.getFieldWithDefault(msg, 2, ""),
    isAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    isActive: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isTenantManager: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isBaseStationManager: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isEndpointManager: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    note: jspb.Message.getFieldWithDefault(msg, 8, ""),
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    firstName: jspb.Message.getFieldWithDefault(msg, 11, ""),
    lastName: jspb.Message.getFieldWithDefault(msg, 12, ""),
    companyName: jspb.Message.getFieldWithDefault(msg, 13, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.User.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.User;
  return proto.kilocenter.api.v1.User.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.User} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.User}
 */
proto.kilocenter.api.v1.User.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAdmin(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsActive(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsTenantManager(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationManager(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointManager(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setNote(value);
      break;
    case 9:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 10:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    case 11:
      var value = /** @type {string} */ (reader.readString());
      msg.setFirstName(value);
      break;
    case 12:
      var value = /** @type {string} */ (reader.readString());
      msg.setLastName(value);
      break;
    case 13:
      var value = /** @type {string} */ (reader.readString());
      msg.setCompanyName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.User.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.User.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.User} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.User.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getIsAdmin();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getIsActive();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsTenantManager();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsBaseStationManager();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsEndpointManager();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getNote();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getFirstName();
  if (f.length > 0) {
    writer.writeString(
      11,
      f
    );
  }
  f = message.getLastName();
  if (f.length > 0) {
    writer.writeString(
      12,
      f
    );
  }
  f = message.getCompanyName();
  if (f.length > 0) {
    writer.writeString(
      13,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string email = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool is_admin = 3;
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.getIsAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setIsAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool is_active = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.getIsActive = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setIsActive = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_tenant_manager = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.getIsTenantManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setIsTenantManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_base_station_manager = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.getIsBaseStationManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setIsBaseStationManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_endpoint_manager = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.getIsEndpointManager = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setIsEndpointManager = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional string note = 8;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getNote = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setNote = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};


/**
 * optional google.protobuf.Timestamp created_at = 9;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.User.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 9));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.User} returns this
*/
proto.kilocenter.api.v1.User.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional google.protobuf.Timestamp updated_at = 10;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.User.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 10));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.User} returns this
*/
proto.kilocenter.api.v1.User.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.User.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 10) != null;
};


/**
 * optional string first_name = 11;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getFirstName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 11, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setFirstName = function(value) {
  return jspb.Message.setProto3StringField(this, 11, value);
};


/**
 * optional string last_name = 12;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getLastName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 12, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setLastName = function(value) {
  return jspb.Message.setProto3StringField(this, 12, value);
};


/**
 * optional string company_name = 13;
 * @return {string}
 */
proto.kilocenter.api.v1.User.prototype.getCompanyName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 13, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.User} returns this
 */
proto.kilocenter.api.v1.User.prototype.setCompanyName = function(value) {
  return jspb.Message.setProto3StringField(this, 13, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateOrganizationRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateOrganizationRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    description: jspb.Message.getFieldWithDefault(msg, 2, ""),
    canHaveBaseStations: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    maxBaseStationCount: jspb.Message.getFieldWithDefault(msg, 4, 0),
    maxEndpointCount: jspb.Message.getFieldWithDefault(msg, 5, 0),
    tagsMap: (f = msg.getTagsMap()) ? f.toObject(includeInstance, undefined) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateOrganizationRequest;
  return proto.kilocenter.api.v1.CreateOrganizationRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateOrganizationRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanHaveBaseStations(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxBaseStationCount(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxEndpointCount(value);
      break;
    case 6:
      var value = msg.getTagsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateOrganizationRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateOrganizationRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDescription();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCanHaveBaseStations();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getMaxBaseStationCount();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getMaxEndpointCount();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getTagsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(6, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string description = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.setDescription = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool can_have_base_stations = 3;
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getCanHaveBaseStations = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.setCanHaveBaseStations = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional int32 max_base_station_count = 4;
 * @return {number}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getMaxBaseStationCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.setMaxBaseStationCount = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional int32 max_endpoint_count = 5;
 * @return {number}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getMaxEndpointCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.setMaxEndpointCount = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * map<string, string> tags = 6;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.getTagsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 6, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationRequest.prototype.clearTagsMap = function() {
  this.getTagsMap().clear();
  return this;};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateOrganizationResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateOrganizationResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    organization: (f = msg.getOrganization()) && proto.kilocenter.api.v1.Organization.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationResponse}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateOrganizationResponse;
  return proto.kilocenter.api.v1.CreateOrganizationResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateOrganizationResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationResponse}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.Organization;
      reader.readMessage(value,proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader);
      msg.setOrganization(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateOrganizationResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateOrganizationResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrganization();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.Organization.serializeBinaryToWriter
    );
  }
};


/**
 * optional Organization organization = 1;
 * @return {?proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.getOrganization = function() {
  return /** @type{?proto.kilocenter.api.v1.Organization} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.Organization, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.Organization|undefined} value
 * @return {!proto.kilocenter.api.v1.CreateOrganizationResponse} returns this
*/
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.setOrganization = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.CreateOrganizationResponse} returns this
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.clearOrganization = function() {
  return this.setOrganization(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateOrganizationResponse.prototype.hasOrganization = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetOrganizationRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetOrganizationRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetOrganizationRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetOrganizationRequest}
 */
proto.kilocenter.api.v1.GetOrganizationRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetOrganizationRequest;
  return proto.kilocenter.api.v1.GetOrganizationRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetOrganizationRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetOrganizationRequest}
 */
proto.kilocenter.api.v1.GetOrganizationRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetOrganizationRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetOrganizationRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetOrganizationRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.GetOrganizationRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.GetOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.GetOrganizationRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetOrganizationResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetOrganizationResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    organization: (f = msg.getOrganization()) && proto.kilocenter.api.v1.Organization.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetOrganizationResponse}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetOrganizationResponse;
  return proto.kilocenter.api.v1.GetOrganizationResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetOrganizationResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetOrganizationResponse}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.Organization;
      reader.readMessage(value,proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader);
      msg.setOrganization(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetOrganizationResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetOrganizationResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrganization();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.Organization.serializeBinaryToWriter
    );
  }
};


/**
 * optional Organization organization = 1;
 * @return {?proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.getOrganization = function() {
  return /** @type{?proto.kilocenter.api.v1.Organization} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.Organization, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.Organization|undefined} value
 * @return {!proto.kilocenter.api.v1.GetOrganizationResponse} returns this
*/
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.setOrganization = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetOrganizationResponse} returns this
 */
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.clearOrganization = function() {
  return this.setOrganization(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetOrganizationResponse.prototype.hasOrganization = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateOrganizationRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    description: jspb.Message.getFieldWithDefault(msg, 3, ""),
    canHaveBaseStations: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    maxBaseStationCount: jspb.Message.getFieldWithDefault(msg, 5, 0),
    maxEndpointCount: jspb.Message.getFieldWithDefault(msg, 6, 0),
    tagsMap: (f = msg.getTagsMap()) ? f.toObject(includeInstance, undefined) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateOrganizationRequest;
  return proto.kilocenter.api.v1.UpdateOrganizationRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanHaveBaseStations(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxBaseStationCount(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxEndpointCount(value);
      break;
    case 7:
      var value = msg.getTagsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateOrganizationRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDescription();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getCanHaveBaseStations();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getMaxBaseStationCount();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getMaxEndpointCount();
  if (f !== 0) {
    writer.writeInt32(
      6,
      f
    );
  }
  f = message.getTagsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(7, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string description = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setDescription = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool can_have_base_stations = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getCanHaveBaseStations = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setCanHaveBaseStations = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional int32 max_base_station_count = 5;
 * @return {number}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getMaxBaseStationCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setMaxBaseStationCount = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional int32 max_endpoint_count = 6;
 * @return {number}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getMaxEndpointCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.setMaxEndpointCount = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};


/**
 * map<string, string> tags = 7;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.getTagsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 7, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationRequest.prototype.clearTagsMap = function() {
  this.getTagsMap().clear();
  return this;};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateOrganizationResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    organization: (f = msg.getOrganization()) && proto.kilocenter.api.v1.Organization.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationResponse}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateOrganizationResponse;
  return proto.kilocenter.api.v1.UpdateOrganizationResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationResponse}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.Organization;
      reader.readMessage(value,proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader);
      msg.setOrganization(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateOrganizationResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrganization();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.Organization.serializeBinaryToWriter
    );
  }
};


/**
 * optional Organization organization = 1;
 * @return {?proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.getOrganization = function() {
  return /** @type{?proto.kilocenter.api.v1.Organization} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.Organization, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.Organization|undefined} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationResponse} returns this
*/
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.setOrganization = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationResponse} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.clearOrganization = function() {
  return this.setOrganization(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationResponse.prototype.hasOrganization = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteOrganizationRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationRequest}
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteOrganizationRequest;
  return proto.kilocenter.api.v1.DeleteOrganizationRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationRequest}
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteOrganizationRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationRequest} returns this
 */
proto.kilocenter.api.v1.DeleteOrganizationRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteOrganizationResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationResponse}
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteOrganizationResponse;
  return proto.kilocenter.api.v1.DeleteOrganizationResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationResponse}
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteOrganizationResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteOrganizationResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.DeleteOrganizationResponse} returns this
 */
proto.kilocenter.api.v1.DeleteOrganizationResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListOrganizationsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListOrganizationsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    pageSize: jspb.Message.getFieldWithDefault(msg, 1, 0),
    pageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    tenantId: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListOrganizationsRequest}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListOrganizationsRequest;
  return proto.kilocenter.api.v1.ListOrganizationsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListOrganizationsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListOrganizationsRequest}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setPageSize(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPageToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTenantId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListOrganizationsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListOrganizationsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPageSize();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTenantId();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
};


/**
 * optional int32 page_size = 1;
 * @return {number}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.getPageSize = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.setPageSize = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.getPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.setPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int64 tenant_id = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.getTenantId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsRequest.prototype.setTenantId = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListOrganizationsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListOrganizationsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    organizationsList: jspb.Message.toObjectList(msg.getOrganizationsList(),
    proto.kilocenter.api.v1.Organization.toObject, includeInstance),
    nextPageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    totalCount: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListOrganizationsResponse;
  return proto.kilocenter.api.v1.ListOrganizationsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListOrganizationsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.Organization;
      reader.readMessage(value,proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader);
      msg.addOrganizations(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTotalCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListOrganizationsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListOrganizationsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrganizationsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.kilocenter.api.v1.Organization.serializeBinaryToWriter
    );
  }
  f = message.getNextPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTotalCount();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
};


/**
 * repeated Organization organizations = 1;
 * @return {!Array<!proto.kilocenter.api.v1.Organization>}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.getOrganizationsList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.Organization>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.Organization, 1));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.Organization>} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse} returns this
*/
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.setOrganizationsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.kilocenter.api.v1.Organization=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.addOrganizations = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.kilocenter.api.v1.Organization, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.clearOrganizationsList = function() {
  return this.setOrganizationsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 total_count = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.getTotalCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationsResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationsResponse.prototype.setTotalCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.Organization.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.Organization.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.Organization} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.Organization.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    tenantId: jspb.Message.getFieldWithDefault(msg, 2, 0),
    name: jspb.Message.getFieldWithDefault(msg, 3, ""),
    description: jspb.Message.getFieldWithDefault(msg, 4, ""),
    state: jspb.Message.getFieldWithDefault(msg, 5, ""),
    externalId: jspb.Message.getFieldWithDefault(msg, 6, ""),
    canHaveBaseStations: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    maxBaseStationCount: jspb.Message.getFieldWithDefault(msg, 8, 0),
    maxEndpointCount: jspb.Message.getFieldWithDefault(msg, 9, 0),
    tagsMap: (f = msg.getTagsMap()) ? f.toObject(includeInstance, undefined) : [],
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.Organization.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.Organization;
  return proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.Organization} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.Organization}
 */
proto.kilocenter.api.v1.Organization.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTenantId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setState(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setExternalId(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanHaveBaseStations(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxBaseStationCount(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxEndpointCount(value);
      break;
    case 10:
      var value = msg.getTagsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 11:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 12:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.Organization.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.Organization.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.Organization} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.Organization.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getTenantId();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getDescription();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getState();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getExternalId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getCanHaveBaseStations();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getMaxBaseStationCount();
  if (f !== 0) {
    writer.writeInt32(
      8,
      f
    );
  }
  f = message.getMaxEndpointCount();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getTagsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(10, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      11,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      12,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.Organization.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int64 tenant_id = 2;
 * @return {number}
 */
proto.kilocenter.api.v1.Organization.prototype.getTenantId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setTenantId = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional string name = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.Organization.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string description = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.Organization.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setDescription = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string state = 5;
 * @return {string}
 */
proto.kilocenter.api.v1.Organization.prototype.getState = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setState = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string external_id = 6;
 * @return {string}
 */
proto.kilocenter.api.v1.Organization.prototype.getExternalId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setExternalId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional bool can_have_base_stations = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.Organization.prototype.getCanHaveBaseStations = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setCanHaveBaseStations = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional int32 max_base_station_count = 8;
 * @return {number}
 */
proto.kilocenter.api.v1.Organization.prototype.getMaxBaseStationCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setMaxBaseStationCount = function(value) {
  return jspb.Message.setProto3IntField(this, 8, value);
};


/**
 * optional int32 max_endpoint_count = 9;
 * @return {number}
 */
proto.kilocenter.api.v1.Organization.prototype.getMaxEndpointCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.setMaxEndpointCount = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * map<string, string> tags = 10;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.kilocenter.api.v1.Organization.prototype.getTagsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 10, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.clearTagsMap = function() {
  this.getTagsMap().clear();
  return this;};


/**
 * optional google.protobuf.Timestamp created_at = 11;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.Organization.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 11));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
*/
proto.kilocenter.api.v1.Organization.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 11, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.Organization.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 11) != null;
};


/**
 * optional google.protobuf.Timestamp updated_at = 12;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.Organization.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 12));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.Organization} returns this
*/
proto.kilocenter.api.v1.Organization.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 12, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.Organization} returns this
 */
proto.kilocenter.api.v1.Organization.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.Organization.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 12) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.AddOrganizationUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    role: jspb.Message.getFieldWithDefault(msg, 3, ""),
    isOrgAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isBaseStationAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isEndpointAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    email: jspb.Message.getFieldWithDefault(msg, 7, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.AddOrganizationUserRequest;
  return proto.kilocenter.api.v1.AddOrganizationUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setRole(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsOrgAdmin(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationAdmin(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointAdmin(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.AddOrganizationUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRole();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getIsOrgAdmin();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsBaseStationAdmin();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsEndpointAdmin();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string role = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getRole = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setRole = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool is_org_admin = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getIsOrgAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setIsOrgAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_base_station_admin = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getIsBaseStationAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setIsBaseStationAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_endpoint_admin = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getIsEndpointAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setIsEndpointAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional string email = 7;
 * @return {string}
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserRequest.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.AddOrganizationUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    member: (f = msg.getMember()) && proto.kilocenter.api.v1.OrganizationUser.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserResponse}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.AddOrganizationUserResponse;
  return proto.kilocenter.api.v1.AddOrganizationUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserResponse}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.OrganizationUser;
      reader.readMessage(value,proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader);
      msg.setMember(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.AddOrganizationUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.AddOrganizationUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMember();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter
    );
  }
};


/**
 * optional OrganizationUser member = 1;
 * @return {?proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.getMember = function() {
  return /** @type{?proto.kilocenter.api.v1.OrganizationUser} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.OrganizationUser, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.OrganizationUser|undefined} value
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserResponse} returns this
*/
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.setMember = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.AddOrganizationUserResponse} returns this
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.clearMember = function() {
  return this.setMember(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.AddOrganizationUserResponse.prototype.hasMember = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetOrganizationUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserRequest}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetOrganizationUserRequest;
  return proto.kilocenter.api.v1.GetOrganizationUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserRequest}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetOrganizationUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.GetOrganizationUserRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetOrganizationUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    member: (f = msg.getMember()) && proto.kilocenter.api.v1.OrganizationUser.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserResponse}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetOrganizationUserResponse;
  return proto.kilocenter.api.v1.GetOrganizationUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserResponse}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.OrganizationUser;
      reader.readMessage(value,proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader);
      msg.setMember(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetOrganizationUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetOrganizationUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMember();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter
    );
  }
};


/**
 * optional OrganizationUser member = 1;
 * @return {?proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.getMember = function() {
  return /** @type{?proto.kilocenter.api.v1.OrganizationUser} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.OrganizationUser, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.OrganizationUser|undefined} value
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserResponse} returns this
*/
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.setMember = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetOrganizationUserResponse} returns this
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.clearMember = function() {
  return this.setMember(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetOrganizationUserResponse.prototype.hasMember = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateOrganizationUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    role: jspb.Message.getFieldWithDefault(msg, 3, ""),
    isOrgAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isBaseStationAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isEndpointAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    updateMask: (f = msg.getUpdateMask()) && google_protobuf_field_mask_pb.FieldMask.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateOrganizationUserRequest;
  return proto.kilocenter.api.v1.UpdateOrganizationUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setRole(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsOrgAdmin(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationAdmin(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointAdmin(value);
      break;
    case 7:
      var value = new google_protobuf_field_mask_pb.FieldMask;
      reader.readMessage(value,google_protobuf_field_mask_pb.FieldMask.deserializeBinaryFromReader);
      msg.setUpdateMask(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateOrganizationUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRole();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getIsOrgAdmin();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsBaseStationAdmin();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsEndpointAdmin();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getUpdateMask();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_field_mask_pb.FieldMask.serializeBinaryToWriter
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string role = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getRole = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setRole = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool is_org_admin = 4;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getIsOrgAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setIsOrgAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_base_station_admin = 5;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getIsBaseStationAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setIsBaseStationAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_endpoint_admin = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getIsEndpointAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setIsEndpointAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional google.protobuf.FieldMask update_mask = 7;
 * @return {?proto.google.protobuf.FieldMask}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.getUpdateMask = function() {
  return /** @type{?proto.google.protobuf.FieldMask} */ (
    jspb.Message.getWrapperField(this, google_protobuf_field_mask_pb.FieldMask, 7));
};


/**
 * @param {?proto.google.protobuf.FieldMask|undefined} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
*/
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.setUpdateMask = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.clearUpdateMask = function() {
  return this.setUpdateMask(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserRequest.prototype.hasUpdateMask = function() {
  return jspb.Message.getField(this, 7) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.UpdateOrganizationUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    member: (f = msg.getMember()) && proto.kilocenter.api.v1.OrganizationUser.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.UpdateOrganizationUserResponse;
  return proto.kilocenter.api.v1.UpdateOrganizationUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.OrganizationUser;
      reader.readMessage(value,proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader);
      msg.setMember(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.UpdateOrganizationUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMember();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter
    );
  }
};


/**
 * optional OrganizationUser member = 1;
 * @return {?proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.getMember = function() {
  return /** @type{?proto.kilocenter.api.v1.OrganizationUser} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.OrganizationUser, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.OrganizationUser|undefined} value
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse} returns this
*/
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.setMember = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.UpdateOrganizationUserResponse} returns this
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.clearMember = function() {
  return this.setMember(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.UpdateOrganizationUserResponse.prototype.hasMember = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.RemoveOrganizationUserRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.RemoveOrganizationUserRequest;
  return proto.kilocenter.api.v1.RemoveOrganizationUserRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.RemoveOrganizationUserRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserRequest} returns this
 */
proto.kilocenter.api.v1.RemoveOrganizationUserRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.RemoveOrganizationUserResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.RemoveOrganizationUserResponse;
  return proto.kilocenter.api.v1.RemoveOrganizationUserResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.RemoveOrganizationUserResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.RemoveOrganizationUserResponse} returns this
 */
proto.kilocenter.api.v1.RemoveOrganizationUserResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListOrganizationUsersRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    status: jspb.Message.getFieldWithDefault(msg, 2, ""),
    pageSize: jspb.Message.getFieldWithDefault(msg, 3, 0),
    pageToken: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListOrganizationUsersRequest;
  return proto.kilocenter.api.v1.ListOrganizationUsersRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setStatus(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setPageSize(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListOrganizationUsersRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getStatus();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPageSize();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getPageToken();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string status = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.getStatus = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.setStatus = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 page_size = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.getPageSize = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.setPageSize = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional string page_token = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.getPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersRequest} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersRequest.prototype.setPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListOrganizationUsersResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    membersList: jspb.Message.toObjectList(msg.getMembersList(),
    proto.kilocenter.api.v1.OrganizationUser.toObject, includeInstance),
    nextPageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    totalCount: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListOrganizationUsersResponse;
  return proto.kilocenter.api.v1.ListOrganizationUsersResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.OrganizationUser;
      reader.readMessage(value,proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader);
      msg.addMembers(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTotalCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListOrganizationUsersResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMembersList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter
    );
  }
  f = message.getNextPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTotalCount();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
};


/**
 * repeated OrganizationUser members = 1;
 * @return {!Array<!proto.kilocenter.api.v1.OrganizationUser>}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.getMembersList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.OrganizationUser>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.OrganizationUser, 1));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.OrganizationUser>} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} returns this
*/
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.setMembersList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.kilocenter.api.v1.OrganizationUser=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.addMembers = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.kilocenter.api.v1.OrganizationUser, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.clearMembersList = function() {
  return this.setMembersList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 total_count = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.getTotalCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListOrganizationUsersResponse} returns this
 */
proto.kilocenter.api.v1.ListOrganizationUsersResponse.prototype.setTotalCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.OrganizationUser.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.OrganizationUser} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.OrganizationUser.toObject = function(includeInstance, msg) {
  var f, obj = {
    orgId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    email: jspb.Message.getFieldWithDefault(msg, 3, ""),
    role: jspb.Message.getFieldWithDefault(msg, 4, ""),
    status: jspb.Message.getFieldWithDefault(msg, 5, ""),
    isOrgAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isBaseStationAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    isEndpointAdmin: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.OrganizationUser.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.OrganizationUser;
  return proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.OrganizationUser} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.OrganizationUser}
 */
proto.kilocenter.api.v1.OrganizationUser.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setEmail(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setRole(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setStatus(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsOrgAdmin(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsBaseStationAdmin(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsEndpointAdmin(value);
      break;
    case 9:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 10:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.OrganizationUser} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.OrganizationUser.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getEmail();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getRole();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getStatus();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getIsOrgAdmin();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsBaseStationAdmin();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getIsEndpointAdmin();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional string org_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string email = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getEmail = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setEmail = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string role = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getRole = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setRole = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string status = 5;
 * @return {string}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getStatus = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setStatus = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional bool is_org_admin = 6;
 * @return {boolean}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getIsOrgAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setIsOrgAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_base_station_admin = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getIsBaseStationAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setIsBaseStationAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional bool is_endpoint_admin = 8;
 * @return {boolean}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getIsEndpointAdmin = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.setIsEndpointAdmin = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional google.protobuf.Timestamp created_at = 9;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 9));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
*/
proto.kilocenter.api.v1.OrganizationUser.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional google.protobuf.Timestamp updated_at = 10;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 10));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
*/
proto.kilocenter.api.v1.OrganizationUser.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.OrganizationUser} returns this
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.OrganizationUser.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 10) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListUserOrganizationsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    userId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsRequest}
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListUserOrganizationsRequest;
  return proto.kilocenter.api.v1.ListUserOrganizationsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsRequest}
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListUserOrganizationsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string user_id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsRequest} returns this
 */
proto.kilocenter.api.v1.ListUserOrganizationsRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListUserOrganizationsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    membershipsList: jspb.Message.toObjectList(msg.getMembershipsList(),
    proto.kilocenter.api.v1.UserMembership.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsResponse}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListUserOrganizationsResponse;
  return proto.kilocenter.api.v1.ListUserOrganizationsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsResponse}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.UserMembership;
      reader.readMessage(value,proto.kilocenter.api.v1.UserMembership.deserializeBinaryFromReader);
      msg.addMemberships(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListUserOrganizationsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListUserOrganizationsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMembershipsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.kilocenter.api.v1.UserMembership.serializeBinaryToWriter
    );
  }
};


/**
 * repeated UserMembership memberships = 1;
 * @return {!Array<!proto.kilocenter.api.v1.UserMembership>}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.getMembershipsList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.UserMembership>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.UserMembership, 1));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.UserMembership>} value
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsResponse} returns this
*/
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.setMembershipsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.kilocenter.api.v1.UserMembership=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.UserMembership}
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.addMemberships = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.kilocenter.api.v1.UserMembership, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.ListUserOrganizationsResponse} returns this
 */
proto.kilocenter.api.v1.ListUserOrganizationsResponse.prototype.clearMembershipsList = function() {
  return this.setMembershipsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateApiKeyRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateApiKeyRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    keyType: jspb.Message.getFieldWithDefault(msg, 2, ""),
    expiresAt: (f = msg.getExpiresAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    organizationId: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateApiKeyRequest;
  return proto.kilocenter.api.v1.CreateApiKeyRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateApiKeyRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setKeyType(value);
      break;
    case 3:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setExpiresAt(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrganizationId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateApiKeyRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateApiKeyRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getKeyType();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getExpiresAt();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getOrganizationId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string key_type = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.getKeyType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.setKeyType = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional google.protobuf.Timestamp expires_at = 3;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.getExpiresAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 3));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest} returns this
*/
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.setExpiresAt = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.clearExpiresAt = function() {
  return this.setExpiresAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.hasExpiresAt = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string organization_id = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.getOrganizationId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyRequest.prototype.setOrganizationId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.CreateApiKeyResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.CreateApiKeyResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    apiKey: (f = msg.getApiKey()) && proto.kilocenter.api.v1.ApiKey.toObject(includeInstance, f),
    rawKey: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyResponse}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.CreateApiKeyResponse;
  return proto.kilocenter.api.v1.CreateApiKeyResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.CreateApiKeyResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyResponse}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.ApiKey;
      reader.readMessage(value,proto.kilocenter.api.v1.ApiKey.deserializeBinaryFromReader);
      msg.setApiKey(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRawKey(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.CreateApiKeyResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.CreateApiKeyResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getApiKey();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.ApiKey.serializeBinaryToWriter
    );
  }
  f = message.getRawKey();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional ApiKey api_key = 1;
 * @return {?proto.kilocenter.api.v1.ApiKey}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.getApiKey = function() {
  return /** @type{?proto.kilocenter.api.v1.ApiKey} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.ApiKey, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.ApiKey|undefined} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyResponse} returns this
*/
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.setApiKey = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.CreateApiKeyResponse} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.clearApiKey = function() {
  return this.setApiKey(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.hasApiKey = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string raw_key = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.getRawKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.CreateApiKeyResponse} returns this
 */
proto.kilocenter.api.v1.CreateApiKeyResponse.prototype.setRawKey = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetApiKeyRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetApiKeyRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetApiKeyRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetApiKeyRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetApiKeyRequest}
 */
proto.kilocenter.api.v1.GetApiKeyRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetApiKeyRequest;
  return proto.kilocenter.api.v1.GetApiKeyRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetApiKeyRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetApiKeyRequest}
 */
proto.kilocenter.api.v1.GetApiKeyRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetApiKeyRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetApiKeyRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetApiKeyRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetApiKeyRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.GetApiKeyRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.GetApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.GetApiKeyRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.GetApiKeyResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.GetApiKeyResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetApiKeyResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    apiKey: (f = msg.getApiKey()) && proto.kilocenter.api.v1.ApiKey.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.GetApiKeyResponse}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.GetApiKeyResponse;
  return proto.kilocenter.api.v1.GetApiKeyResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.GetApiKeyResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.GetApiKeyResponse}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.ApiKey;
      reader.readMessage(value,proto.kilocenter.api.v1.ApiKey.deserializeBinaryFromReader);
      msg.setApiKey(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.GetApiKeyResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.GetApiKeyResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.GetApiKeyResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getApiKey();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.kilocenter.api.v1.ApiKey.serializeBinaryToWriter
    );
  }
};


/**
 * optional ApiKey api_key = 1;
 * @return {?proto.kilocenter.api.v1.ApiKey}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.getApiKey = function() {
  return /** @type{?proto.kilocenter.api.v1.ApiKey} */ (
    jspb.Message.getWrapperField(this, proto.kilocenter.api.v1.ApiKey, 1));
};


/**
 * @param {?proto.kilocenter.api.v1.ApiKey|undefined} value
 * @return {!proto.kilocenter.api.v1.GetApiKeyResponse} returns this
*/
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.setApiKey = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.GetApiKeyResponse} returns this
 */
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.clearApiKey = function() {
  return this.setApiKey(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.GetApiKeyResponse.prototype.hasApiKey = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteApiKeyRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyRequest}
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteApiKeyRequest;
  return proto.kilocenter.api.v1.DeleteApiKeyRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyRequest}
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteApiKeyRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyRequest} returns this
 */
proto.kilocenter.api.v1.DeleteApiKeyRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.DeleteApiKeyResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyResponse}
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.DeleteApiKeyResponse;
  return proto.kilocenter.api.v1.DeleteApiKeyResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyResponse}
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.DeleteApiKeyResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.DeleteApiKeyResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.DeleteApiKeyResponse} returns this
 */
proto.kilocenter.api.v1.DeleteApiKeyResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListApiKeysRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListApiKeysRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListApiKeysRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    pageSize: jspb.Message.getFieldWithDefault(msg, 1, 0),
    pageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListApiKeysRequest}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListApiKeysRequest;
  return proto.kilocenter.api.v1.ListApiKeysRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListApiKeysRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListApiKeysRequest}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setPageSize(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setPageToken(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListApiKeysRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListApiKeysRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListApiKeysRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPageSize();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional int32 page_size = 1;
 * @return {number}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.getPageSize = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysRequest} returns this
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.setPageSize = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.getPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysRequest} returns this
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.setPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string user_id = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysRequest} returns this
 */
proto.kilocenter.api.v1.ListApiKeysRequest.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.kilocenter.api.v1.ListApiKeysResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ListApiKeysResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ListApiKeysResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListApiKeysResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    apiKeysList: jspb.Message.toObjectList(msg.getApiKeysList(),
    proto.kilocenter.api.v1.ApiKey.toObject, includeInstance),
    nextPageToken: jspb.Message.getFieldWithDefault(msg, 2, ""),
    totalCount: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ListApiKeysResponse;
  return proto.kilocenter.api.v1.ListApiKeysResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ListApiKeysResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.kilocenter.api.v1.ApiKey;
      reader.readMessage(value,proto.kilocenter.api.v1.ApiKey.deserializeBinaryFromReader);
      msg.addApiKeys(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTotalCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ListApiKeysResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ListApiKeysResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ListApiKeysResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getApiKeysList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.kilocenter.api.v1.ApiKey.serializeBinaryToWriter
    );
  }
  f = message.getNextPageToken();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTotalCount();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
};


/**
 * repeated ApiKey api_keys = 1;
 * @return {!Array<!proto.kilocenter.api.v1.ApiKey>}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.getApiKeysList = function() {
  return /** @type{!Array<!proto.kilocenter.api.v1.ApiKey>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.kilocenter.api.v1.ApiKey, 1));
};


/**
 * @param {!Array<!proto.kilocenter.api.v1.ApiKey>} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse} returns this
*/
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.setApiKeysList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.kilocenter.api.v1.ApiKey=} opt_value
 * @param {number=} opt_index
 * @return {!proto.kilocenter.api.v1.ApiKey}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.addApiKeys = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.kilocenter.api.v1.ApiKey, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse} returns this
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.clearApiKeysList = function() {
  return this.setApiKeysList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse} returns this
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 total_count = 3;
 * @return {number}
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.getTotalCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.kilocenter.api.v1.ListApiKeysResponse} returns this
 */
proto.kilocenter.api.v1.ListApiKeysResponse.prototype.setTotalCount = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.kilocenter.api.v1.ApiKey.prototype.toObject = function(opt_includeInstance) {
  return proto.kilocenter.api.v1.ApiKey.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.kilocenter.api.v1.ApiKey} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ApiKey.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    orgId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 3, ""),
    name: jspb.Message.getFieldWithDefault(msg, 4, ""),
    keyPrefix: jspb.Message.getFieldWithDefault(msg, 5, ""),
    keyType: jspb.Message.getFieldWithDefault(msg, 6, ""),
    isActive: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    expiresAt: (f = msg.getExpiresAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    lastUsedAt: (f = msg.getLastUsedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.kilocenter.api.v1.ApiKey}
 */
proto.kilocenter.api.v1.ApiKey.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.kilocenter.api.v1.ApiKey;
  return proto.kilocenter.api.v1.ApiKey.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.kilocenter.api.v1.ApiKey} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.kilocenter.api.v1.ApiKey}
 */
proto.kilocenter.api.v1.ApiKey.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrgId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setKeyPrefix(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setKeyType(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsActive(value);
      break;
    case 8:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setExpiresAt(value);
      break;
    case 9:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setLastUsedAt(value);
      break;
    case 10:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.kilocenter.api.v1.ApiKey.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.kilocenter.api.v1.ApiKey.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.kilocenter.api.v1.ApiKey} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.kilocenter.api.v1.ApiKey.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOrgId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getKeyPrefix();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getKeyType();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getIsActive();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getExpiresAt();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getLastUsedAt();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string org_id = 2;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getOrgId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setOrgId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string user_id = 3;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string name = 4;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string key_prefix = 5;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getKeyPrefix = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setKeyPrefix = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string key_type = 6;
 * @return {string}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getKeyType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setKeyType = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional bool is_active = 7;
 * @return {boolean}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getIsActive = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.setIsActive = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional google.protobuf.Timestamp expires_at = 8;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getExpiresAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 8));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
*/
proto.kilocenter.api.v1.ApiKey.prototype.setExpiresAt = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.clearExpiresAt = function() {
  return this.setExpiresAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.ApiKey.prototype.hasExpiresAt = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional google.protobuf.Timestamp last_used_at = 9;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getLastUsedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 9));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
*/
proto.kilocenter.api.v1.ApiKey.prototype.setLastUsedAt = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.clearLastUsedAt = function() {
  return this.setLastUsedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.ApiKey.prototype.hasLastUsedAt = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional google.protobuf.Timestamp created_at = 10;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.kilocenter.api.v1.ApiKey.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 10));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
*/
proto.kilocenter.api.v1.ApiKey.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.kilocenter.api.v1.ApiKey} returns this
 */
proto.kilocenter.api.v1.ApiKey.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.kilocenter.api.v1.ApiKey.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 10) != null;
};


goog.object.extend(exports, proto.kilocenter.api.v1);
