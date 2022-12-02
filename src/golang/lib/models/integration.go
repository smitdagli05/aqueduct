package models

import (
	"strings"
	"time"

	"github.com/aqueducthq/aqueduct/lib/models/shared"
	"github.com/google/uuid"
)

const (
	IntegrationTable = "integration"

	// Integration table column names
	IntegrationID        = "id"
	IntegrationOrgID     = "organization_id"
	IntegrationUserID    = "user_id"
	IntegrationService   = "service"
	IntegrationName      = "name"
	IntegrationConfig    = "config"
	IntegrationCreatedAt = "created_at"
	IntegrationValidated = "validated"
)

// A Integration maps to the integration table.
type Integration struct {
	ID        uuid.UUID                `db:"id" json:"id"`
	UserID    uuid.NullUUID            `db:"user_id" json:"user_id"`
	OrgID     string                   `db:"organization_id"`
	Service   shared.Service           `db:"service"`
	Name      string                   `db:"name"`
	Config    shared.IntegrationConfig `db:"config"`
	CreatedAt time.Time                `db:"created_at"`
	Validated bool                     `db:"validated"`
}

// IntegrationCols returns a comma-separated string of all Integration columns.
func IntegrationCols() string {
	return strings.Join(allIntegrationCols(), ",")
}

func allIntegrationCols() []string {
	return []string{
		IntegrationID,
		IntegrationOrgID,
		IntegrationUserID,
		IntegrationService,
		IntegrationName,
		IntegrationConfig,
		IntegrationCreatedAt,
		IntegrationValidated,
	}
}