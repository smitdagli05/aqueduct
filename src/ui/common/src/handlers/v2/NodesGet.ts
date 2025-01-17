// This file should map exactly to
// src/golang/cmd/server/handler/v2/nodes_get.go

import { APIKeyParameter } from '../parameters/Header';
import { DagIdParameter, WorkflowIdParameter } from '../parameters/Path';
import { NodesResponse } from '../responses/Node';

export type NodesGetRequest = APIKeyParameter &
  DagIdParameter &
  WorkflowIdParameter;

export type NodesGetResponse = NodesResponse;

export const nodesGetQuery = (req: NodesGetRequest) => ({
  url: `workflow/${req.workflowId}/dag/${req.dagId}/nodes`,
  headers: { 'api-key': req.apiKey },
});
