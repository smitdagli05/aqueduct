import {
  faCalendar,
  faEllipsis,
  faMicrochip,
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Collapse, Tooltip, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import React, { useEffect, useLayoutEffect, useState } from 'react';
import Markdown from 'react-markdown';
import { useSelector } from 'react-redux';
import { visitParents } from 'unist-util-visit-parents';

import { RootState } from '../../stores/store';
import style from '../../styles/markdown.module.css';
import { theme } from '../../styles/theme/theme';
import { getNextUpdateTime } from '../../utils/cron';
import { EngineType } from '../../utils/engine';
import ExecutionStatus, { LoadingStatusEnum } from '../../utils/shared';
import {
  getWorkflowEngineTypes,
  WorkflowDag,
  WorkflowUpdateTrigger,
} from '../../utils/workflows';
import ResourceItem from '../pages/workflows/components/ResourceItem';
import VersionSelector from './version_selector';
import { StatusIndicator } from './workflowStatus';

export const WorkflowPageContentId = 'workflow-page-main';

type Props = {
  workflowDag: WorkflowDag;
};

const ContainerWidthBreakpoint = 700;

const WorkflowHeader: React.FC<Props> = ({ workflowDag }) => {
  const currentNode = useSelector(
    (state: RootState) => state.nodeSelectionReducer.selected
  );

  // NOTE: The 1000 here is just a placeholder. By the time the page snaps into place,
  // it will be overridden.
  const [containerWidth, setContainerWidth] = useState(1000);
  const narrowView = containerWidth < ContainerWidthBreakpoint;

  const getContainerSize = () => {
    const container = document.getElementById(WorkflowPageContentId);

    if (!container) {
      // The page hasn't fully rendered yet.
      setContainerWidth(1000); // Just a default value.
    } else {
      setContainerWidth(container.clientWidth);
    }
  };

  // TODO (ENG-2302): useLayoutEffect here. May want to figure out some way to debounce as it gets called quite quickly when resizing.
  window.addEventListener('resize', getContainerSize);
  useLayoutEffect(getContainerSize, [currentNode]);

  const [showDescription, setShowDescription] = useState(false);
  const workflow = useSelector((state: RootState) => state.workflowReducer);
  const workflowHistory = useSelector(
    (state: RootState) => state.workflowHistoryReducer
  );
  const [selectedResultIdx, setSelectedResultIdx] = useState(0);

  // Whenever the workflow reducer's selected result changes, we update our local state to
  // have the correct index of the workflow history list. This is used to ensure that we show
  // the correct execution status in the header.
  useEffect(() => {
    if (workflowHistory.status.loading !== LoadingStatusEnum.Succeeded) {
      return;
    }

    for (let i = 0; i < workflowHistory.history.versions.length; i++) {
      const result = workflowHistory.history.versions[i];
      if (result.versionId === workflow.selectedResult.id) {
        setSelectedResultIdx(i);
      }
    }
  }, [workflow.selectedResult]);

  let selectedWorkflowStatus = ExecutionStatus.Unknown;
  if (workflowHistory.status.loading === LoadingStatusEnum.Succeeded) {
    selectedWorkflowStatus =
      workflowHistory.history.versions[selectedResultIdx]?.exec_state.status;
  }

  const name = workflowDag.metadata?.name ?? '';
  const description = workflowDag.metadata?.description;

  let nextUpdate;
  if (
    workflowDag.metadata?.schedule?.trigger ===
      WorkflowUpdateTrigger.Periodic &&
    !workflowDag.metadata?.schedule?.paused
  ) {
    const nextUpdateTime = getNextUpdateTime(
      workflowDag.metadata?.schedule?.cron_schedule
    );

    nextUpdate = nextUpdateTime.toDate().toLocaleString();
  }

  const showAirflowUpdateWarning =
    workflowDag.engine_config.type === EngineType.Airflow &&
    !workflowDag.engine_config.airflow_config?.matches_airflow;
  const airflowUpdateWarning = (
    <Box maxWidth="800px">
      <Alert severity="warning">
        Please copy the latest Airflow DAG file to your Airflow server if you
        have not done so already. New Airflow DAG runs will not be synced
        properly with Aqueduct until you have copied the file.
      </Alert>
    </Box>
  );

  /**
   * Wrap text in a `custom-typography` tag
   */
  function rehypeWrapText() {
    return function wrapTextTransform(tree) {
      visitParents(tree, 'text', (node, ancestors) => {
        if (ancestors[ancestors.length - 1]?.tagName !== 'custom-typography') {
          node.type = 'element';
          node.tagName = 'custom-typography';
          node.children = [{ type: 'text', value: node.value }];
        }
      });
    };
  }

  const engines = getWorkflowEngineTypes(workflowDag);

  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: narrowView ? 'start' : 'center',
          flexDirection: narrowView ? 'column' : 'row',
          transition: 'height 100ms',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          {!!workflow.dagResults && workflow.dagResults.length > 0 && (
            <StatusIndicator status={selectedWorkflowStatus} />
          )}

          <Typography
            variant="h5"
            sx={{ ml: 1, lineHeight: 1 }}
            fontWeight="normal"
          >
            {name}
          </Typography>

          {workflow.dagResults && workflow.dagResults.length > 0 && (
            <Box ml={2}>
              <VersionSelector />
            </Box>
          )}
        </Box>

        <Box
          sx={{
            fontSize: '20px',
            p: 1,
            ml: narrowView ? 0 : 1,
            mt: narrowView ? 1 : 0,
            borderRadius: '8px',
            ':hover': {
              backgroundColor: theme.palette.gray[50],
            },
            cursor: 'pointer',
          }}
          onClick={() => setShowDescription(!showDescription)}
        >
          <Tooltip title="See more" arrow>
            <FontAwesomeIcon
              icon={faEllipsis}
              style={{
                transform: showDescription ? 'rotateX(180deg)' : '',
                transition: 'transform 200ms',
              }}
            />
          </Tooltip>
        </Box>
      </Box>

      <Collapse in={showDescription}>
        <Box display="flex" alignItems="center" my={1}>
          {/* Display the Workflow Engine. */}
          <Tooltip title={'Compute Engine(s)'} arrow>
            <Box display="flex" alignItems="center">
              <FontAwesomeIcon
                icon={faMicrochip}
                color={theme.palette.gray[800]}
              />
              <Box display="flex" flexDirection="row">
                {engines.map((engine) => (
                  <Box ml={1} key={engine}>
                    <ResourceItem resource={engine} />
                  </Box>
                ))}
              </Box>
            </Box>
          </Tooltip>
          {/* Display the next workflow run. */}
          {nextUpdate && (
            <Tooltip title="Next Workflow Run" arrow>
              <Box display="flex" alignItems="center" ml={2}>
                <Box mr={1}>
                  <FontAwesomeIcon
                    icon={faCalendar}
                    color={theme.palette.gray[800]}
                  />
                </Box>
                <Typography>{nextUpdate}</Typography>
              </Box>
            </Tooltip>
          )}
        </Box>

        <Box
          sx={{
            backgroundColor: theme.palette.gray[25],
            px: 2,
            py: 1,
            my: 1,
            maxWidth: '800px',
            borderRadius: '4px',
          }}
        >
          <Markdown
            className={style.reactMarkdown}
            rehypePlugins={[rehypeWrapText]}
            components={{
              'custom-typography': ({ children }) => (
                <Typography variant="body1">{children}</Typography>
              ),
            }}
          >
            {description === '' ? '*No description.*' : description}
          </Markdown>
        </Box>
      </Collapse>

      {showAirflowUpdateWarning && airflowUpdateWarning}
    </Box>
  );
};

export default WorkflowHeader;
