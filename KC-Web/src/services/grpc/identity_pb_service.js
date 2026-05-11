// package: kilocenter.api.v1
// file: identity.proto

var identity_pb = require("./identity_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var IdentityService = (function () {
  function IdentityService() {}
  IdentityService.serviceName = "kilocenter.api.v1.IdentityService";
  return IdentityService;
})();

IdentityService.Login = {
  methodName: "Login",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.LoginRequest,
  responseType: identity_pb.LoginResponse,
};

IdentityService.RefreshTokens = {
  methodName: "RefreshTokens",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RefreshTokensRequest,
  responseType: identity_pb.RefreshTokensResponse,
};

IdentityService.GetProfile = {
  methodName: "GetProfile",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetProfileRequest,
  responseType: identity_pb.GetProfileResponse,
};

IdentityService.GetAuthSettings = {
  methodName: "GetAuthSettings",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetAuthSettingsRequest,
  responseType: identity_pb.GetAuthSettingsResponse,
};

IdentityService.Logout = {
  methodName: "Logout",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.LogoutRequest,
  responseType: identity_pb.LogoutResponse,
};

IdentityService.ChangePassword = {
  methodName: "ChangePassword",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ChangePasswordRequest,
  responseType: identity_pb.ChangePasswordResponse,
};

IdentityService.ExchangeOIDC = {
  methodName: "ExchangeOIDC",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ExchangeOIDCRequest,
  responseType: identity_pb.LoginResponse,
};

IdentityService.ExchangeOAuth2 = {
  methodName: "ExchangeOAuth2",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ExchangeOAuth2Request,
  responseType: identity_pb.LoginResponse,
};

IdentityService.RegisterAccount = {
  methodName: "RegisterAccount",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RegisterAccountRequest,
  responseType: identity_pb.LoginResponse,
};

IdentityService.CreateUser = {
  methodName: "CreateUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateUserRequest,
  responseType: identity_pb.CreateUserResponse,
};

IdentityService.GetUser = {
  methodName: "GetUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetUserRequest,
  responseType: identity_pb.GetUserResponse,
};

IdentityService.UpdateUser = {
  methodName: "UpdateUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateUserRequest,
  responseType: identity_pb.UpdateUserResponse,
};

IdentityService.DeleteUser = {
  methodName: "DeleteUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteUserRequest,
  responseType: identity_pb.DeleteUserResponse,
};

IdentityService.ListUsers = {
  methodName: "ListUsers",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListUsersRequest,
  responseType: identity_pb.ListUsersResponse,
};

IdentityService.UpdateUserPassword = {
  methodName: "UpdateUserPassword",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateUserPasswordRequest,
  responseType: identity_pb.UpdateUserPasswordResponse,
};

IdentityService.CreateOrganization = {
  methodName: "CreateOrganization",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateOrganizationRequest,
  responseType: identity_pb.CreateOrganizationResponse,
};

IdentityService.GetOrganization = {
  methodName: "GetOrganization",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetOrganizationRequest,
  responseType: identity_pb.GetOrganizationResponse,
};

IdentityService.UpdateOrganization = {
  methodName: "UpdateOrganization",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateOrganizationRequest,
  responseType: identity_pb.UpdateOrganizationResponse,
};

IdentityService.DeleteOrganization = {
  methodName: "DeleteOrganization",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteOrganizationRequest,
  responseType: identity_pb.DeleteOrganizationResponse,
};

IdentityService.ListOrganizations = {
  methodName: "ListOrganizations",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListOrganizationsRequest,
  responseType: identity_pb.ListOrganizationsResponse,
};

IdentityService.AddOrganizationUser = {
  methodName: "AddOrganizationUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.AddOrganizationUserRequest,
  responseType: identity_pb.AddOrganizationUserResponse,
};

IdentityService.GetOrganizationUser = {
  methodName: "GetOrganizationUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetOrganizationUserRequest,
  responseType: identity_pb.GetOrganizationUserResponse,
};

IdentityService.UpdateOrganizationUser = {
  methodName: "UpdateOrganizationUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateOrganizationUserRequest,
  responseType: identity_pb.UpdateOrganizationUserResponse,
};

IdentityService.RemoveOrganizationUser = {
  methodName: "RemoveOrganizationUser",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RemoveOrganizationUserRequest,
  responseType: identity_pb.RemoveOrganizationUserResponse,
};

IdentityService.ListOrganizationUsers = {
  methodName: "ListOrganizationUsers",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListOrganizationUsersRequest,
  responseType: identity_pb.ListOrganizationUsersResponse,
};

IdentityService.ListUserOrganizations = {
  methodName: "ListUserOrganizations",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListUserOrganizationsRequest,
  responseType: identity_pb.ListUserOrganizationsResponse,
};

IdentityService.CreateApiKey = {
  methodName: "CreateApiKey",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateApiKeyRequest,
  responseType: identity_pb.CreateApiKeyResponse,
};

IdentityService.GetApiKey = {
  methodName: "GetApiKey",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetApiKeyRequest,
  responseType: identity_pb.GetApiKeyResponse,
};

IdentityService.DeleteApiKey = {
  methodName: "DeleteApiKey",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteApiKeyRequest,
  responseType: identity_pb.DeleteApiKeyResponse,
};

IdentityService.ListApiKeys = {
  methodName: "ListApiKeys",
  service: IdentityService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListApiKeysRequest,
  responseType: identity_pb.ListApiKeysResponse,
};

exports.IdentityService = IdentityService;

function IdentityServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

IdentityServiceClient.prototype.login = function login(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.Login, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.refreshTokens = function refreshTokens(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.RefreshTokens, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.getProfile = function getProfile(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.GetProfile, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.getAuthSettings = function getAuthSettings(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.GetAuthSettings, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.logout = function logout(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.Logout, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.changePassword = function changePassword(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ChangePassword, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.exchangeOIDC = function exchangeOIDC(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ExchangeOIDC, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.exchangeOAuth2 = function exchangeOAuth2(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ExchangeOAuth2, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.registerAccount = function registerAccount(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.RegisterAccount, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.createUser = function createUser(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.CreateUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.getUser = function getUser(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.GetUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.updateUser = function updateUser(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.UpdateUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.deleteUser = function deleteUser(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.DeleteUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.listUsers = function listUsers(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ListUsers, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.updateUserPassword =
  function updateUserPassword(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.UpdateUserPassword, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.createOrganization =
  function createOrganization(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.CreateOrganization, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.getOrganization = function getOrganization(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.GetOrganization, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.updateOrganization =
  function updateOrganization(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.UpdateOrganization, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.deleteOrganization =
  function deleteOrganization(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.DeleteOrganization, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.listOrganizations = function listOrganizations(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ListOrganizations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.addOrganizationUser =
  function addOrganizationUser(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.AddOrganizationUser, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.getOrganizationUser =
  function getOrganizationUser(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.GetOrganizationUser, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.updateOrganizationUser =
  function updateOrganizationUser(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.UpdateOrganizationUser, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.removeOrganizationUser =
  function removeOrganizationUser(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.RemoveOrganizationUser, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.listOrganizationUsers =
  function listOrganizationUsers(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.ListOrganizationUsers, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.listUserOrganizations =
  function listUserOrganizations(requestMessage, metadata, callback) {
    if (arguments.length === 2) {
      callback = arguments[1];
    }
    var client = grpc.unary(IdentityService.ListUserOrganizations, {
      request: requestMessage,
      host: this.serviceHost,
      metadata: metadata,
      transport: this.options.transport,
      debug: this.options.debug,
      onEnd: function (response) {
        if (callback) {
          if (response.status !== grpc.Code.OK) {
            var err = new Error(response.statusMessage);
            err.code = response.status;
            err.metadata = response.trailers;
            callback(err, null);
          } else {
            callback(null, response.message);
          }
        }
      },
    });
    return {
      cancel: function () {
        callback = null;
        client.close();
      },
    };
  };

IdentityServiceClient.prototype.createApiKey = function createApiKey(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.CreateApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.getApiKey = function getApiKey(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.GetApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.deleteApiKey = function deleteApiKey(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.DeleteApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

IdentityServiceClient.prototype.listApiKeys = function listApiKeys(
  requestMessage,
  metadata,
  callback,
) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(IdentityService.ListApiKeys, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

exports.IdentityServiceClient = IdentityServiceClient;
