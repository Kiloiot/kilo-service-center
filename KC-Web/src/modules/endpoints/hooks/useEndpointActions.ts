/**
 * The attach, detach and delete actions of the end point page; each reports
 * its outcome through the shared feedback.
 */

import {
  useAttachEndpoint,
  useDeleteEndpoint,
  useDetachEndpoint,
} from "@hooks";

import { useFeedback } from "@contexts/feedback";
import { logger } from "@utils/logger";
import {
  ENDPOINT_FORM,
  ERR_ATTACH_ENDPOINT,
  ERR_DELETE_ENDPOINT,
  ERR_DETACH_ENDPOINT,
  LOG_ATTACH_FAILED,
  LOG_DELETE_FAILED,
  LOG_DETACH_FAILED,
  LOG_PRE_DELETE_DETACH_FAILED,
} from "@constants/messages";

export function useEndpointActions(epEui: string) {
  const feedback = useFeedback();
  const attachMutation = useAttachEndpoint();
  const detachMutation = useDetachEndpoint();
  const deleteMutation = useDeleteEndpoint();

  const attach = async () => {
    try {
      await attachMutation.mutateAsync(epEui);
      feedback.success(ENDPOINT_FORM.MSG_ENDPOINT_ATTACHED);
    } catch (err) {
      logger.error(LOG_ATTACH_FAILED, err);
      feedback.error(ERR_ATTACH_ENDPOINT);
    }
  };

  const detach = async () => {
    try {
      await detachMutation.mutateAsync(epEui);
      feedback.success(ENDPOINT_FORM.MSG_ENDPOINT_DETACHED);
    } catch (err) {
      logger.error(LOG_DETACH_FAILED, err);
      feedback.error(ERR_DETACH_ENDPOINT);
    }
  };

  // An attached end point is detached first; a failed detach does not block the delete.
  const remove = async (attached: boolean, onDeleted: () => void) => {
    try {
      if (attached) {
        await detachMutation
          .mutateAsync(epEui)
          .catch((err) => logger.warn(LOG_PRE_DELETE_DETACH_FAILED, err));
      }
      await deleteMutation.mutateAsync(epEui);
      onDeleted();
      feedback.success(ENDPOINT_FORM.MSG_ENDPOINT_DELETED);
    } catch (err) {
      logger.error(LOG_DELETE_FAILED, err);
      feedback.error(ERR_DELETE_ENDPOINT);
    }
  };

  return {
    attach,
    detach,
    remove,
    isAttaching: attachMutation.isPending,
    isDetaching: detachMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}
