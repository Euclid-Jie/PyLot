package db

import "database/sql"

func ListRecentRuns(limit int) ([]RecentRun, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := DB.Query(`
		SELECT record_id,target_id,target_type,target_name,status,started_at,ended_at,is_error,trigger_source,schedule_id
		FROM (
			SELECT rr.id AS record_id,
				rr.script_id AS target_id,
				'script' AS target_type,
				COALESCE(s.name, '脚本 #' || CAST(rr.script_id AS TEXT)) AS target_name,
				rr.status,
				rr.started_at,
				rr.ended_at,
				rr.is_error,
				COALESCE(rr.trigger_source, 'unknown') AS trigger_source,
				COALESCE(rr.schedule_id, 0) AS schedule_id
			FROM run_records rr
			LEFT JOIN scripts s ON s.id=rr.script_id
			WHERE COALESCE(rr.trigger_source, 'unknown')<>'workflow'
				AND NOT EXISTS (
					SELECT 1 FROM workflow_run_nodes wrn WHERE wrn.run_record_id=rr.id
				)

			UNION ALL

			SELECT wr.id AS record_id,
				wr.workflow_id AS target_id,
				'workflow' AS target_type,
				COALESCE(w.name, '工作流 #' || CAST(wr.workflow_id AS TEXT)) AS target_name,
				wr.status,
				wr.started_at,
				wr.ended_at,
				CASE WHEN wr.status IN ('error','timeout','killed') THEN 1 ELSE 0 END AS is_error,
				COALESCE(wr.trigger_source, 'unknown') AS trigger_source,
				COALESCE(wr.schedule_id, 0) AS schedule_id
			FROM workflow_runs wr
			LEFT JOIN workflows w ON w.id=wr.workflow_id
		)
		ORDER BY started_at DESC,target_type,record_id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]RecentRun, 0, limit)
	for rows.Next() {
		var run RecentRun
		var endedAt sql.NullTime
		if err := rows.Scan(
			&run.RecordID,
			&run.TargetID,
			&run.TargetType,
			&run.TargetName,
			&run.Status,
			&run.StartedAt,
			&endedAt,
			&run.IsError,
			&run.TriggerSource,
			&run.ScheduleID,
		); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			run.EndedAt = &endedAt.Time
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}
