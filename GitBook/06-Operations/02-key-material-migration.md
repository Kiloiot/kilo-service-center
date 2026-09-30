# Key material migration (rekey)

KiloCenter stores every secret key column as an authenticated AES-256-GCM
envelope under the deployment's `KILOCENTER_MASTER_KEY`. Databases created by
older releases can still hold earlier encodings: cleartext keys, ciphertext
from the retired passphrase-based scheme, and rows in the retired
`endpoint_keys` tables. The `rekey` command converts all of them in place, and
newer releases require it: the runtime readers accept only valid envelopes,
so KC-Core, KC-Identity and `migrate -command=up` stop at schema `000143` until
the conversion is complete.

The command ships in the KC-Core image as `/usr/local/bin/rekey` and covers:

- `endpoints.nwk_key`, `endpoints.app_key`
- `endpoint_sessions.session_key`
- `messages.nwk_sn_key`, `messages_archive.nwk_sn_key`
- `basestations.tls_key`
- `bssci_pending_operations.metadata` (`encryptedKey`)
- `bssci_pending_operations.operation_data` (`nwkSnKey` of recovery records
  written before keys were kept out of them; the key moves into the encrypted
  `encryptedKey` metadata)
- `endpoint_keys` / `endpoint_keys_archive` reconciliation and encrypted export

Every conversion runs in its own transaction and is verified by a
decrypt-and-compare read-back before it commits. The command is idempotent: a
clean database is a no-op. Output contains only counts and keyed digests,
never key material.

## What happens if the conversion is skipped

Upgrading a database from schema `000142` or earlier stops at `000143`, the
first version whose endpoint key columns fit an envelope. If any surface above
still holds a value that is not an envelope, or a row remains in the retired
`endpoint_keys` tables, the migration run ends with an error that names the
surfaces and asks for `rekey`. The schema is left clean at `000143`, so run
the command as described below and start the services again. Fresh databases
and databases that are already converted pass the check without any extra
step, and the check never runs again once the schema is past `000143`.

## Modes

| Mode | Effect |
|---|---|
| `-mode=dry-run` | Classify every value and print counts. Changes nothing. |
| `-mode=apply` | Convert legacy and cleartext values, reconcile `endpoint_keys`. |
| `-mode=verify` | Exit nonzero unless every surface holds only valid envelopes and the `endpoint_keys` tables are gone or empty. |

## Legacy decryption keys

Rows written by the retired scheme are encrypted under SHA-256 of a
passphrase: the deployment's `KC_ENCRYPTION_KEY`, or, when that variable was
never set, a development passphrase that is published in the repository. The
command never guesses which one applies:

- `-allow-published-development-key` decrypts rows sealed under the published
  development passphrase. **Every deployment that never set
  `KC_ENCRYPTION_KEY` needs it**, and that includes every Docker Compose and
  Helm installation up to v1.3.0, because neither ever set the variable. Those
  rows were never secret - anyone with the database and the public repository
  could read them - so converting them loses nothing; the conversion protects
  them from now on. Rotate any key that must stay confidential.
- `-legacy-passphrase-env NAME` names an environment variable holding your own
  passphrase, for a deployment that did set `KC_ENCRYPTION_KEY`.

The two options are mutually exclusive. Without the matching one, legacy
ciphertext is counted as `legacy_locked`, `apply` exits nonzero and `verify`
fails.

## Kubernetes (Helm)

The chart runs a `pre-upgrade` hook before the new pods (and therefore
before their startup migrations): it first migrates the schema to `000143`
(`migrate -command=up -to=143`), the first version whose endpoint key columns
accept envelopes, and then runs `rekey -mode=apply`. It is enabled by default;
see the `rekey` block in `values.yaml`.

- The hook reads `secrets.masterKey`, `postgresql.password` and
  `rekey.legacyPassphrase` from its own Secret, created for the upgrade hooks
  and deleted by Helm once they succeed: the release Secret still holds the
  previous release's values while the hooks run.
- `rekey.allowPublishedDevelopmentKey` is `true` by default, because the chart
  never set `KC_ENCRYPTION_KEY`. If your deployment set one anyway, pass it as
  `rekey.legacyPassphrase` and set `rekey.allowPublishedDevelopmentKey=false`.
- If the retired `endpoint_keys` tables hold rows the hook cannot fold into
  the endpoints table, or any archive rows, `apply` needs somewhere to write
  the encrypted export. Point `rekey.archiveExport.existingClaim` at an
  existing PersistentVolumeClaim, writable by uid/gid 1000; each upgrade
  writes `endpoint-keys-<release revision>.enc` under
  `rekey.archiveExport.mountPath`. Without a claim the hook fails when such
  rows exist.

## Docker Compose / manual

If `.env` has no `KILOCENTER_MASTER_KEY` yet (installations up to v1.3.0 did
not need one), generate it first with `openssl rand -hex 32`, add it to
`.env` and keep it with your other secrets: every stored key is encrypted
under it from now on, and losing it makes them unreadable.

Stop the KC-Core and KC-Identity services first (both run migrations at
start-up; the compose services are `kilocenter` and `kc-identity`), take a
database backup, build the new images, migrate the schema to `000143` (the
rekeyed envelopes do not fit the 16-byte key checks of earlier versions), run
the command against the database, then start the stack again. `-to` never
migrates down, so repeating the step is safe. The `kilocenter` service already
receives `KILOCENTER_MASTER_KEY` from `.env`; `DB_PASSWORD` is the
`KILOCENTER_POSTGRESQL_PASSWORD` from the same file. The commands below use
`-allow-published-development-key` because Docker Compose installations up to
v1.3.0 never set `KC_ENCRYPTION_KEY`:

```bash
docker compose stop kc-gateway kilocenter kc-identity
docker compose build

docker compose run --rm --no-deps \
  -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_NAME=kilocenter -e DB_USER=kilocenter -e DB_PASSWORD=... \
  --entrypoint migrate kilocenter -command=up -to=143

docker compose run --rm --no-deps \
  -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_NAME=kilocenter -e DB_USER=kilocenter -e DB_PASSWORD=... \
  --entrypoint rekey kilocenter -mode=dry-run -allow-published-development-key

# review the counts, then:
docker compose run --rm --no-deps \
  -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_NAME=kilocenter -e DB_USER=kilocenter -e DB_PASSWORD=... \
  --entrypoint rekey kilocenter -mode=apply -allow-published-development-key

docker compose run --rm --no-deps \
  -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_NAME=kilocenter -e DB_USER=kilocenter -e DB_PASSWORD=... \
  --entrypoint rekey kilocenter -mode=verify -allow-published-development-key

docker compose up -d
```

If `apply` reports that it needs an export location, mount a host directory
writable by uid 1000 and name a file in it, for example
`-v "$PWD/rekey-export:/export"` on the `docker compose run` line and
`-archive-export=/export/endpoint-keys.enc` after `-mode=apply`.

## endpoint_keys reconciliation

`apply` resolves the retired `endpoint_keys` tables row by row:

- a row matching its endpoint's stored key is redundant and deleted;
- a row whose endpoint has no key yet is written into the endpoint (as an
  envelope, read back and compared) and deleted;
- rows that conflict with a different stored key, reference a missing
  endpoint, or carry a key type with no destination column are exported and
  reported as conflicts - they are never dropped silently;
- archive rows are exported and deleted.

Exports are written to the path given by `-archive-export` as a single
master-key envelope. Archive rows are deleted once exported, so point the
path at storage that outlives the container that runs the command. Conflicted
rows must be resolved by an operator (using the export for reference) before
the schema can move past `000143`.
