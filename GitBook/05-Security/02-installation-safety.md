# Before you install KiloCenter Community Edition

If you are evaluating KiloCenter, use an isolated machine or virtual machine and test data. Keep the
example services away from public networks and other users. CHIRP USA LLC maintains this software.
The current examples are not an approved production installation procedure.

## What needs attention

The repository includes reusable administrator, database and MQTT credentials and signing-key values.
A person who can reach an installation that still uses those values could gain access they should
not have. Signing keys protect login tokens; they are separate from user passwords. Some example
ports are published on the host, so opening `localhost` in your browser does not make the services
local-only. These are verified source/default issues; no customer compromise is claimed here.

Changing the administrator password is necessary for an existing default account, but does not
replace unique service credentials, unpredictable installation-owned signing keys, private internal
ports, verified encrypted connections and a tested recovery process. The complete correction is in
progress. Until it is delivered and verified, do not use the examples for customer data or expose them
to the Internet. If you already run them, restrict access, preserve your data and review the settings
with the person responsible for your installation. Do not delete the database as a security remedy.

## Choose the software deliberately

A release tag such as `v1.3.0` names a release; `latest` can change without identifying the same bytes.
The July 26 release workflow published images at `ghcr.io/kiloiot/kc-core`, `kc-gateway`, `kc-identity`
and `kc-web`. Current chart defaults use a different nested repository path. Check every application
image and the certificate-generator image before installation. Registry access also needs verification;
a repository-name correction alone does not grant access or complete release security checks.

The existing release records do not yet provide a fully verified installation package, complete
built-component list, update/recovery record and signature verification. Do not interpret the presence
of source code, a build badge or a release tag as proof that these checks passed.

## Preserve data during maintenance

For a hostname change or routine server certificate renewal, follow
[certificate-only renewal](03-certificate-renewal.md). Keep the existing certificate authority (CA),
which is the trusted issuer used by connected stations. Do not run `docker compose down -v` for this
purpose: it removes data volumes, including the database. Do not clear all Redis data to troubleshoot
one login error. Preserve the error and diagnose the affected service first.

## Report a problem

For a suspected security weakness, email **info@kiloiot.de** and identify KiloCenter Community Edition,
the version if known, and what happened. Do not publish passwords, private keys, tokens or customer
data in an issue. See the [security policy](../../SECURITY.md) for reporting instructions and
[Support](../../SUPPORT.md) for ordinary installation questions.

This notice concerns the Community Edition. It does not make a CRA conformity, certification,
commercial-support or guaranteed-response-time claim.
