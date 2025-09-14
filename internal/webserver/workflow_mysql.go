// Fetch workflow data from db similar to:
// See: https://github.com/argoproj/argo-workflows/blob/main/persist/sqldb/workflow_archive.go
package webserver

import (
	"database/sql"
	"encoding/json"
	"fmt"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflowscommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	_ "github.com/go-sql-driver/mysql"
)

func getWorkflowsFromMysql(workflowMysqlAddr string) ([]workflowsv1.Workflow, error) {
	db, err := sql.Open("mysql", workflowMysqlAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql db %s: %s", workflowMysqlAddr, err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT workflow from argo_archived_workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to query mysql db %s: %s", workflowMysqlAddr, err)
	}
	defer rows.Close()

	var workflows []workflowsv1.Workflow
	for rows.Next() {

		var workflowBytes sql.RawBytes
		err = rows.Scan(&workflowBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read mysql query %s: %s", workflowMysqlAddr, err)
		}

		var workflow workflowsv1.Workflow
		err = json.Unmarshal(workflowBytes, &workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal workflow %s: %s", workflowMysqlAddr, err)
		}

		workflow.Labels[workflowscommon.LabelKeyWorkflowArchivingStatus] = "Persisted"
		workflows = append(workflows, workflow)
	}

	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close mysql query %s: %s", workflowMysqlAddr, err)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in mysql query %s: %s", workflowMysqlAddr, err)
	}

	return workflows, nil
}
