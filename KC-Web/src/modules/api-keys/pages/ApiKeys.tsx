/**
 * API Keys Page
 *
 * Management page for API keys used for programmatic access.
 * Allows users to list, create, and delete API keys.
 */

import React, { useState } from "react";
import { Navigate } from "react-router-dom";

import { Alert, Box } from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { useSession } from "@contexts/SessionContext";
import { useApiKeys } from "@hooks/useApiKeys";
import { useCapabilities } from "@hooks/useCapabilities";
import { API_KEY_PAGE_SIZE, ROUTES } from "@constants/app";
import {
  ERR_LOAD_API_KEYS,
  MSG_API_KEY_COPIED,
  MSG_API_KEY_CREATED,
} from "@constants/messages";

import ApiKeyRevealDialog from "../components/ApiKeyRevealDialog";
import ApiKeysHeader from "../components/ApiKeysHeader";
import ApiKeysTable from "../components/ApiKeysTable";
import CreateApiKeyDialog from "../components/CreateApiKeyDialog";
import DeleteApiKeyDialog from "../components/DeleteApiKeyDialog";
import { useKeyDeletion } from "../hooks";

const ApiKeys: React.FC = () => {
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();

  // Runtime admin guard (deep link protection)
  if (isHydrated && !isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (!isHydrated) {
    return null;
  }

  return <ApiKeysContent />;
};

const ApiKeysContent: React.FC = () => {
  const feedback = useFeedback();
  const [createOpen, setCreateOpen] = useState(false);
  const [rawKey, setRawKey] = useState("");
  const { data, isLoading, isError } = useApiKeys({
    pageSize: API_KEY_PAGE_SIZE,
  });
  const deletion = useKeyDeletion();

  const handleCreated = (key: string) => {
    setCreateOpen(false);
    setRawKey(key);
    feedback.success(MSG_API_KEY_CREATED);
  };

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      <ApiKeysHeader onCreate={() => setCreateOpen(true)} />
      {isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {ERR_LOAD_API_KEYS}
        </Alert>
      )}
      <ApiKeysTable
        apiKeys={data?.apiKeys ?? []}
        isLoading={isLoading}
        onDelete={deletion.request}
      />
      <CreateApiKeyDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={handleCreated}
      />
      <ApiKeyRevealDialog
        rawKey={rawKey}
        onCopied={() => feedback.success(MSG_API_KEY_COPIED)}
        onClose={() => setRawKey("")}
      />
      <DeleteApiKeyDialog deletion={deletion} />
    </Box>
  );
};

export default ApiKeys;
