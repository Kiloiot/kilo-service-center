# Renew a server certificate without deleting your data

The installation operator uses this procedure when the server certificate is nearing expiry or its
hostname must change. Keep the existing certificate authority (CA) and its private key. A routine
server renewal does not require deleting Docker volumes, the database or station registrations.

Use a maintenance window: base station and application-center connections will pause when KC-Core
stops. First try this procedure on an isolated copy of your installation. The certificate tool's
server-only behaviour has been tested; every installed container version and recovery arrangement
must still be checked by its operator.

## Before changing files

1. Confirm the installation, existing image and intended hostname. Use the same Compose project and
   files as the running installation. Do not change images as part of this procedure.
2. Make a restricted, recoverable backup of the existing certificate directory and confirm that the
   CA certificate, CA private key, server certificate and server private key are present. Preserve
   file ownership and permissions. Keep private keys out of Git, tickets and shared logs. Confirm
   the backup can be restored before proceeding.
3. Confirm you have `openssl` available to verify the result. If the CA or backup is missing, stop.
   Do not run initial certificate generation to work around a missing CA.

## Docker Compose installation

Run from the repository root, using the installation's normal Compose options. Replace
`your-hostname.example.com` with the hostname stations will actually use.

1. Stop the service that loads the server certificate:

   ```bash
   docker compose stop kilocenter
   ```

2. Renew only the server certificate and key in the existing certificate volume:

   ```bash
   docker compose run --rm --no-deps --pull never --entrypoint certgen certgen \
     -dir /app/certificates -server-only -server your-hostname.example.com -days 365
   ```

   The explicit entrypoint matters: the normal `certgen` service starts a shell for first-install
   checks. If this command fails, leave KC-Core stopped and follow the failure steps below.

3. Copy only the public CA and server certificates into a temporary verification directory using
   `docker compose cp kilocenter:/app/certificates/ca.crt` and the corresponding `server.crt` path,
   each followed by your chosen local destination. Compare the CA certificate with the backup;
   it must be unchanged. Verify the new server certificate:

   ```bash
   openssl verify -CAfile ca.crt -verify_hostname your-hostname.example.com server.crt
   ```

4. Only after verification succeeds, start KC-Core:

   ```bash
   docker compose start kilocenter
   ```

5. Check service health and confirm that a known station and, if used, application center reconnect.
   Keep the maintenance time, hostname, public certificate fingerprint and result. Update the
   configured hostname for future maintenance. Remove temporary public-certificate copies.

## Linux-host installation

Stop KC-Core with the installation's normal service manager and back up the certificate directory.
Run its existing `certgen` tool with the same certificate directory and `-server-only`, as above.
Verify the unchanged CA and new hostname before restarting KC-Core. Then check reconnection and
retain the result. Do not regenerate the CA.

## If renewal or reconnection fails

Keep KC-Core stopped. Restore the backed-up server certificate **and its matching private key** with
the original ownership and permissions. Verify that they match and that the CA is unchanged, then
restart and check the previous connection. If the old certificate has expired, its key is compromised,
or the backup is unavailable, do not restore service with invalid trust or disable verification.
Keep the service isolated and obtain installation support. Do not reset data volumes.

A compromised or expired CA requires a separate trust-replacement procedure and coordinated station
updates. This server-only procedure is not suitable for that case.
