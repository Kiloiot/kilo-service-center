-- Fold the legacy basestation_certificates table (019) into the canonical
-- basestations.tls_* columns and drop it.
--
-- The certificate service reads and writes tls_ca_certificate,
-- tls_certificate, tls_key, tls_cert_fingerprint and tls_cert_expires_at on
-- basestations; no code path ever read basestation_certificates. An active
-- legacy CA PEM fills an empty tls_ca_certificate, an active legacy client
-- certificate fills an empty tls_certificate together with its fingerprint
-- and expiry. When both stores already hold a certificate they must agree
-- (same fingerprint or same PEM), otherwise the migration aborts.
--
-- Legacy private keys are never copied: private_key_encrypted is raw BYTEA
-- from a different cipher, while tls_key holds the application key-envelope
-- string. A legacy private key whose base station has no canonical tls_key
-- aborts the migration so the operator can re-issue the certificate through
-- the service. Only non-secret history metadata is archived to
-- system_events; PEM bodies and key material are never archived.

DO $$
DECLARE
    orphan_key_count bigint;
    ca_conflict_count bigint;
    client_conflict_count bigint;
    source_count bigint;
    archived_count bigint;
BEGIN
    SELECT count(*) INTO orphan_key_count
    FROM basestation_certificates c
    JOIN basestations b ON b.id = c.basestation_id
    WHERE c.certificate_type = 'client'
      AND c.is_active
      AND c.private_key_encrypted IS NOT NULL
      AND b.tls_key IS NULL;

    IF orphan_key_count > 0 THEN
        RAISE EXCEPTION 'basestation_certificates holds % active client key(s) with no canonical basestations.tls_key; re-issue those certificates before re-running', orphan_key_count;
    END IF;

    SELECT count(*) INTO ca_conflict_count
    FROM basestation_certificates c
    JOIN basestations b ON b.id = c.basestation_id
    WHERE c.certificate_type = 'ca'
      AND c.is_active
      AND b.tls_ca_certificate IS NOT NULL
      AND btrim(b.tls_ca_certificate) <> btrim(c.certificate_pem);

    IF ca_conflict_count > 0 THEN
        RAISE EXCEPTION 'basestation_certificates has % active CA certificate(s) that differ from basestations.tls_ca_certificate; reconcile them before re-running', ca_conflict_count;
    END IF;

    SELECT count(*) INTO client_conflict_count
    FROM basestation_certificates c
    JOIN basestations b ON b.id = c.basestation_id
    WHERE c.certificate_type = 'client'
      AND c.is_active
      AND b.tls_certificate IS NOT NULL
      AND btrim(b.tls_certificate) <> btrim(c.certificate_pem)
      AND b.tls_cert_fingerprint IS DISTINCT FROM c.fingerprint;

    IF client_conflict_count > 0 THEN
        RAISE EXCEPTION 'basestation_certificates has % active client certificate(s) that differ from basestations.tls_certificate; reconcile them before re-running', client_conflict_count;
    END IF;

    UPDATE basestations b
    SET tls_ca_certificate = c.certificate_pem
    FROM basestation_certificates c
    WHERE c.basestation_id = b.id
      AND c.certificate_type = 'ca'
      AND c.is_active
      AND b.tls_ca_certificate IS NULL;

    UPDATE basestations b
    SET tls_certificate = c.certificate_pem,
        tls_cert_fingerprint = COALESCE(b.tls_cert_fingerprint, c.fingerprint),
        tls_cert_expires_at = COALESCE(b.tls_cert_expires_at, c.not_after)
    FROM basestation_certificates c
    WHERE c.basestation_id = b.id
      AND c.certificate_type = 'client'
      AND c.is_active
      AND b.tls_certificate IS NULL;

    SELECT count(*) INTO source_count FROM basestation_certificates;

    WITH archived AS (
        INSERT INTO system_events (
            tenant_id, event_type, event_category, severity,
            source_type, source_name, title, description,
            basestation_id, data, occurred_at, recorded_at, status
        )
        SELECT
            b.tenant_id,
            'basestation.certificate.archived',
            'basestation',
            'info',
            'migration',
            encode(b.bs_eui, 'hex'),
            format('Archived %s certificate record %s', c.certificate_type, c.id),
            format('Legacy basestation_certificates row %s archived by migration 000149 (active=%s)', c.id, c.is_active),
            c.basestation_id,
            jsonb_build_object(
                'legacyTable', 'basestation_certificates',
                'legacyId', c.id,
                'certificateType', c.certificate_type,
                'fingerprint', c.fingerprint,
                'subjectDn', c.subject_dn,
                'issuerDn', c.issuer_dn,
                'notBefore', c.not_before,
                'notAfter', c.not_after,
                'isActive', c.is_active,
                'hadPrivateKey', c.private_key_encrypted IS NOT NULL,
                'createdAt', c.created_at,
                'updatedAt', c.updated_at
            ),
            COALESCE(c.created_at, NOW()),
            NOW(),
            'resolved'
        FROM basestation_certificates c
        JOIN basestations b ON b.id = c.basestation_id
        RETURNING 1
    )
    SELECT count(*) INTO archived_count FROM archived;

    IF archived_count <> source_count THEN
        RAISE EXCEPTION 'basestation_certificates archive mismatch: % source row(s), % archived', source_count, archived_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_basestation_certificates_expiry;
DROP INDEX IF EXISTS idx_basestation_certificates_active;

DROP TABLE IF EXISTS basestation_certificates;
