// This file should mirror src/golang/workflow/artifact/response.go
import { OperatorType } from 'src/utils/operators';

import { ArtifactType } from '../../utils/artifacts';
import { ExecState } from '../../utils/shared';

export type ArtifactResponse = {
  id: string;
  name: string;
  description: string;
  type: ArtifactType;
  from: string;
  to: string[];
};

export type ArtifactResultMetadataResponse = {
  id: string;
  content_path: string;
  serialization_type: string;
  content_serialized?: string;
  exec_state?: ExecState;
};

export type ArtifactResultResponse = ArtifactResponse & {
  result?: ArtifactResultMetadataResponse;
  operatorType?: OperatorType;
};

export type ListArtifactResultsResponse = {
  results: ArtifactResultMetadataResponse[];
};