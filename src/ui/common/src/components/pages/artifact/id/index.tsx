import { CircularProgress } from '@mui/material';
import Box from '@mui/material/Box';
import React from 'react';

import {
  DagResultResponse,
  getMetricsAndChecksOnArtifact,
} from '../../../../handlers/responses/dagDeprecated';
import UserProfile from '../../../../utils/auth';
import { OperatorType } from '../../../../utils/operators';
import ExecutionStatus from '../../../../utils/shared';
import DefaultLayout, { SidesheetContentWidth } from '../../../layouts/default';
import CsvExporter from '../../../workflows/artifact/csvExporter';
import {
  ChecksOverview,
  MetricsOverview,
} from '../../../workflows/artifact/metricsAndChecksOverview';
import OperatorSummaryList from '../../../workflows/operator/summaryList';
import RequireDagOrResult from '../../../workflows/RequireDagOrResult';
import DetailsPageHeader from '../../components/DetailsPageHeader';
import { LayoutProps } from '../../types';
import useWorkflow from '../../workflow/id/hook';
import Preview from './components/Preview';
import useArtifact from './hook';

type ArtifactDetailsPageProps = {
  user: UserProfile;
  Layout?: React.FC<LayoutProps>;
  workflowIdProp?: string;
  workflowDagIdProp?: string;
  workflowDagResultIdProp?: string;
  artifactIdProp?: string;
  sideSheetMode?: boolean;
};

const ArtifactDetailsPage: React.FC<ArtifactDetailsPageProps> = ({
  user,
  Layout = DefaultLayout,
  workflowIdProp,
  workflowDagIdProp,
  workflowDagResultIdProp,
  artifactIdProp,
  sideSheetMode = false,
}) => {
  const {
    breadcrumbs: wfBreadcrumbs,
    workflowId,
    workflowDagId,
    workflowDagResultId,
    workflowDagWithLoadingStatus,
    workflowDagResultWithLoadingStatus,
  } = useWorkflow(
    user.apiKey,
    workflowIdProp,
    workflowDagIdProp,
    workflowDagResultIdProp
  );

  const { breadcrumbs, artifact, contentWithLoadingStatus } = useArtifact(
    user.apiKey,
    artifactIdProp,
    wfBreadcrumbs,
    workflowDagResultId,
    workflowDagWithLoadingStatus,
    workflowDagResultWithLoadingStatus,
    !sideSheetMode
  );

  const dagResult =
    workflowDagResultWithLoadingStatus?.result ??
    (workflowDagWithLoadingStatus?.result as DagResultResponse);
  const { metrics, checks } = dagResult
    ? getMetricsAndChecksOnArtifact(dagResult, artifact.id)
    : { metrics: [], checks: [] };

  if (!artifact) {
    return (
      <Layout breadcrumbs={breadcrumbs} user={user}>
        <CircularProgress />
      </Layout>
    );
  }

  const mapOperators = (opIds: string[]) =>
    opIds
      .map((opId) => (dagResult?.operators ?? {})[opId])
      .filter((op) => !!op && op.spec?.type !== OperatorType.Param);

  const inputs = mapOperators([artifact.from]);
  const outputs = mapOperators(artifact.to ? artifact.to : []);

  let upstreamPending = false;
  inputs.some((operator) => {
    const operator_pending =
      operator?.result?.exec_state?.status === ExecutionStatus.Pending;
    if (operator_pending) {
      upstreamPending = operator_pending;
    }
    return operator_pending;
  });

  const artifactStatus = artifact?.result?.exec_state?.status;
  const previewAvailable =
    artifactStatus && artifactStatus !== ExecutionStatus.Canceled;

  return (
    <Layout breadcrumbs={breadcrumbs} user={user}>
      <RequireDagOrResult
        dagWithLoadingStatus={workflowDagWithLoadingStatus}
        dagResultWithLoadingStatus={workflowDagResultWithLoadingStatus}
      >
        <Box width={sideSheetMode ? SidesheetContentWidth : '100%'}>
          <Box width="100%">
            {!sideSheetMode && (
              <Box width="100%" display="flex" alignItems="center">
                <DetailsPageHeader
                  name={artifact ? artifact.name : 'Artifact'}
                  status={artifactStatus}
                />
                <CsvExporter
                  artifact={artifact}
                  contentWithLoadingStatus={contentWithLoadingStatus}
                />
              </Box>
            )}

            <Box
              display="flex"
              width="100%"
              mt={sideSheetMode ? '16px' : '64px'}
            >
              {inputs.length > 0 && (
                <Box width="100%" mr="32px">
                  <OperatorSummaryList
                    title={'Generated By'}
                    workflowId={workflowId}
                    dagId={workflowDagId}
                    dagResultId={workflowDagResultId}
                    operatorResults={inputs}
                  />
                </Box>
              )}

              {outputs.length > 0 && (
                <Box width="100%">
                  <OperatorSummaryList
                    title={'Consumed By'}
                    workflowId={workflowId}
                    dagId={workflowDagId}
                    dagResultId={workflowDagResultId}
                    operatorResults={outputs}
                  />
                </Box>
              )}
            </Box>

            {workflowDagResultWithLoadingStatus && (
              <Preview
                upstreamPending={upstreamPending}
                previewAvailable={previewAvailable}
                artifact={artifact}
                contentWithLoadingStatus={contentWithLoadingStatus}
              />
            )}

            <Box display="flex" width="100%">
              <MetricsOverview metrics={metrics} />
              <Box width="96px" />
              <ChecksOverview checks={checks} />
            </Box>
          </Box>
        </Box>
      </RequireDagOrResult>
    </Layout>
  );
};

export default ArtifactDetailsPage;
