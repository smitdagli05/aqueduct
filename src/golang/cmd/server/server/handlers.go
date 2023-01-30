package server

import (
	"github.com/aqueducthq/aqueduct/cmd/server/handler"
	"github.com/aqueducthq/aqueduct/cmd/server/routes"
)

func (s *AqServer) Handlers() map[string]handler.Handler {
	return map[string]handler.Handler{
		routes.ArchiveNotificationRoute: &handler.ArchiveNotificationHandler{
			Database: s.Database,

			NotificationRepo: s.NotificationRepo,
		},
		routes.ConnectIntegrationRoute: &handler.ConnectIntegrationHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			IntegrationRepo:    s.IntegrationRepo,
			OperatorRepo:       s.OperatorRepo,

			PauseServer:   s.Pause,
			RestartServer: s.Restart,
		},
		routes.DeleteIntegrationRoute: &handler.DeleteIntegrationHandler{
			Database: s.Database,
			Vault:    s.Vault,

			DAGRepo:                  s.DAGRepo,
			ExecutionEnvironmentRepo: s.ExecutionEnvironmentRepo,
			IntegrationRepo:          s.IntegrationRepo,
			OperatorRepo:             s.OperatorRepo,
		},
		routes.DeleteWorkflowRoute: &handler.DeleteWorkflowHandler{
			Database:   s.Database,
			Engine:     s.AqEngine,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo:          s.IntegrationRepo,
			ExecutionEnvironmentRepo: s.ExecutionEnvironmentRepo,
			OperatorRepo:             s.OperatorRepo,
			WorkflowRepo:             s.WorkflowRepo,
		},
		routes.EditIntegrationRoute: &handler.EditIntegrationHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.EditWorkflowRoute: &handler.EditWorkflowHandler{
			Database: s.Database,
			Engine:   s.AqEngine,

			ArtifactRepo: s.ArtifactRepo,
			DAGRepo:      s.DAGRepo,
			DAGEdgeRepo:  s.DAGEdgeRepo,
			OperatorRepo: s.OperatorRepo,
			WorkflowRepo: s.WorkflowRepo,
		},
		routes.ExportFunctionRoute: &handler.ExportFunctionHandler{
			Database: s.Database,

			DAGRepo:      s.DAGRepo,
			OperatorRepo: s.OperatorRepo,
		},
		routes.GetArtifactResultRoute: &handler.GetArtifactResultHandler{
			Database: s.Database,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			DAGResultRepo:      s.DAGResultRepo,
		},
		routes.GetArtifactVersionsRoute: &handler.GetArtifactVersionsHandler{
			Database: s.Database,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			DAGResultRepo:      s.DAGResultRepo,
			OperatorRepo:       s.OperatorRepo,
			OperatorResultRepo: s.OperatorResultRepo,
		},
		routes.GetNodePositionsRoute: &handler.GetNodePositionsHandler{},
		routes.GetOperatorResultRoute: &handler.GetOperatorResultHandler{
			Database: s.Database,

			DAGResultRepo:      s.DAGResultRepo,
			OperatorRepo:       s.OperatorRepo,
			OperatorResultRepo: s.OperatorResultRepo,
		},
		routes.GetUserProfileRoute: &handler.GetUserProfileHandler{},
		routes.ListWorkflowObjectsRoute: &handler.ListWorkflowObjectsHandler{
			Database: s.Database,

			OperatorRepo: s.OperatorRepo,
			WorkflowRepo: s.WorkflowRepo,
		},
		routes.GetWorkflowRoute: &handler.GetWorkflowHandler{
			Database: s.Database,
			Vault:    s.Vault,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			DAGEdgeRepo:        s.DAGEdgeRepo,
			DAGResultRepo:      s.DAGResultRepo,
			OperatorRepo:       s.OperatorRepo,
			OperatorResultRepo: s.OperatorResultRepo,
			WorkflowRepo:       s.WorkflowRepo,
		},
		routes.GetWorkflowDagResultRoute: &handler.GetWorkflowDagResultHandler{
			Database: s.Database,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			DAGEdgeRepo:        s.DAGEdgeRepo,
			DAGResultRepo:      s.DAGResultRepo,
			OperatorRepo:       s.OperatorRepo,
			OperatorResultRepo: s.OperatorResultRepo,
			WorkflowRepo:       s.WorkflowRepo,
		},
		routes.ListArtifactResultsRoute: &handler.ListArtifactResultsHandler{
			Database: s.Database,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
		},
		routes.ListIntegrationsRoute: &handler.ListIntegrationsHandler{
			Database: s.Database,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.ListNotificationsRoute: &handler.ListNotificationsHandler{
			Database: s.Database,

			DAGResultRepo:    s.DAGResultRepo,
			NotificationRepo: s.NotificationRepo,
		},
		routes.ListOperatorsForIntegrationRoute: &handler.ListOperatorsForIntegrationHandler{
			Database: s.Database,

			DAGRepo:         s.DAGRepo,
			IntegrationRepo: s.IntegrationRepo,
			OperatorRepo:    s.OperatorRepo,
		},
		routes.ListWorkflowsRoute: &handler.ListWorkflowsHandler{
			Database: s.Database,
			Vault:    s.Vault,

			ArtifactRepo:       s.ArtifactRepo,
			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGRepo:            s.DAGRepo,
			DAGEdgeRepo:        s.DAGEdgeRepo,
			DAGResultRepo:      s.DAGResultRepo,
			OperatorRepo:       s.OperatorRepo,
			OperatorResultRepo: s.OperatorResultRepo,
			WorkflowRepo:       s.WorkflowRepo,
		},
		routes.PreviewTableRoute: &handler.PreviewTableHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.PreviewRoute: &handler.PreviewHandler{
			Database:      s.Database,
			GithubManager: s.GithubManager,
			AqEngine:      s.AqEngine,

			ExecutionEnvironmentRepo: s.ExecutionEnvironmentRepo,
			IntegrationRepo:          s.IntegrationRepo,
		},
		routes.DiscoverRoute: &handler.DiscoverHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
			OperatorRepo:    s.OperatorRepo,
		},
		routes.ListIntegrationObjectsRoute: &handler.ListIntegrationObjectsHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.CreateTableRoute: &handler.CreateTableHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.RefreshWorkflowRoute: &handler.RefreshWorkflowHandler{
			Database: s.Database,
			Engine:   s.AqEngine,
			Vault:    s.Vault,

			WorkflowRepo: s.WorkflowRepo,
		},
		routes.RegisterWorkflowRoute: &handler.RegisterWorkflowHandler{
			Database:      s.Database,
			JobManager:    s.JobManager,
			GithubManager: s.GithubManager,
			Vault:         s.Vault,
			Engine:        s.AqEngine,

			ArtifactRepo:             s.ArtifactRepo,
			DAGRepo:                  s.DAGRepo,
			DAGEdgeRepo:              s.DAGEdgeRepo,
			ExecutionEnvironmentRepo: s.ExecutionEnvironmentRepo,
			IntegrationRepo:          s.IntegrationRepo,
			OperatorRepo:             s.OperatorRepo,
			WatcherRepo:              s.WatcherRepo,
			WorkflowRepo:             s.WorkflowRepo,
		},
		routes.RegisterAirflowWorkflowRoute: &handler.RegisterAirflowWorkflowHandler{
			RegisterWorkflowHandler: handler.RegisterWorkflowHandler{
				Database:      s.Database,
				JobManager:    s.JobManager,
				GithubManager: s.GithubManager,
				Vault:         s.Vault,

				ArtifactRepo:    s.ArtifactRepo,
				DAGRepo:         s.DAGRepo,
				DAGEdgeRepo:     s.DAGEdgeRepo,
				IntegrationRepo: s.IntegrationRepo,
				OperatorRepo:    s.OperatorRepo,
				WatcherRepo:     s.WatcherRepo,
				WorkflowRepo:    s.WorkflowRepo,
			},

			ArtifactResultRepo: s.ArtifactResultRepo,
			DAGResultRepo:      s.DAGResultRepo,
			OperatorResultRepo: s.OperatorResultRepo,
		},
		routes.ResetApiKeyRoute: &handler.ResetApiKeyHandler{
			Database: s.Database,
			UserRepo: s.UserRepo,
		},
		routes.TestIntegrationRoute: &handler.TestIntegrationHandler{
			Database:   s.Database,
			JobManager: s.JobManager,
			Vault:      s.Vault,

			IntegrationRepo: s.IntegrationRepo,
		},
		routes.GetServerVersionRoute:     &handler.GetServerVersionHandler{},
		routes.GetServerEnvironmentRoute: &handler.GetServerEnvironmentHandler{},
	}
}
