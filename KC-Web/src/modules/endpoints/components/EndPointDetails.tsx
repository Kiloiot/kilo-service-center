/**
 * End point page: the overview with its actions above named tabs (Activity,
 * Downlink, Traffic, Configuration).
 */

import React, { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useEndpoint } from "@hooks";
import { Card, CardContent, Divider } from "@mui/material";

import {
  ENDPOINT_ATTACH_STATUS,
  ENDPOINT_DETAIL_LAYOUT,
  ROUTES,
} from "@constants/app";

import { useBlueprintInfo, useEndpointActions } from "../hooks";
import EditEndPointDialog from "./EditEndPointDialog";
import { EndpointDeleteDialog } from "./EndpointDeleteDialog";
import { EndpointDetailTabs } from "./EndpointDetailTabs";
import {
  type EndPointInput,
  EndpointOverviewPanel,
} from "./EndpointOverviewPanel";

interface EndPointDetailsProps {
  endPoint: EndPointInput;
  onDelete?: (id: string) => void;
}

const EndPointDetails: React.FC<EndPointDetailsProps> = ({
  endPoint,
  onDelete,
}) => {
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const actions = useEndpointActions(endPoint.epEui);

  const { data: endpointDetails } = useEndpoint(endPoint.epEui);
  const endpointData = endpointDetails || endPoint;
  const blueprintInfo = useBlueprintInfo(endpointDetails?.deviceModelId);
  const attached =
    endpointData.attachStatus === ENDPOINT_ATTACH_STATUS.ATTACHED;

  const handleDelete = () => {
    setDeleteDialogOpen(false);
    return actions.remove(attached, () =>
      onDelete ? onDelete(endPoint.epEui) : navigate(ROUTES.ENDPOINTS),
    );
  };

  return (
    <>
      <Card sx={{ mt: 2, mb: 2 }}>
        <CardContent>
          <EndpointOverviewPanel
            endPoint={endPoint}
            endpointData={endpointData}
            blueprintInfo={blueprintInfo}
            isAttaching={actions.isAttaching}
            isDetaching={actions.isDetaching}
            onAttach={actions.attach}
            onDetach={actions.detach}
            onEdit={() => setEditDialogOpen(true)}
            onDeleteClick={() => setDeleteDialogOpen(true)}
          />
          <Divider sx={{ my: ENDPOINT_DETAIL_LAYOUT.DIVIDER_MARGIN_Y }} />
          <EndpointDetailTabs
            epEui={endPoint.epEui}
            endpoint={endpointDetails}
          />
        </CardContent>
      </Card>

      <EndpointDeleteDialog
        open={deleteDialogOpen}
        name={endPoint.name || endPoint.epEui}
        attached={attached}
        pending={actions.isDeleting}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={handleDelete}
      />

      <EditEndPointDialog
        open={editDialogOpen}
        onClose={() => setEditDialogOpen(false)}
        endpoint={endpointDetails ?? null}
      />
    </>
  );
};

export default EndPointDetails;
