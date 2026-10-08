# Third-Party Notices

KiloCenter Community Edition is licensed under the GNU Affero General Public
License v3.0 or later (see [LICENSE](LICENSE)). It contains the following code
adapted from third-party projects, which remains under its original license.

## grpc-proxy

- Project: [github.com/mwitkow/grpc-proxy](https://github.com/mwitkow/grpc-proxy)
- Copyright 2017 Michal Witkowski
- License: Apache License, Version 2.0
  ([full text](https://www.apache.org/licenses/LICENSE-2.0))
- Used in: `KC-Gateway/internal/proxy/stream_handler.go`, adapted from
  `proxy/handler.go` (the transparent stream handler that forwards calls to
  KC-Core). The adapted file states its origin in its header.
