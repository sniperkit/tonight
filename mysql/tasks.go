package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/bobinette/tonight"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *tonight.Task) error {
	if t.ID != 0 {
		return errors.New("cannot update a task")
	}

	row := r.db.QueryRowContext(ctx, "SELECT max(rank) FROM tasks WHERE deleted = ? AND done = ?", false, false)
	var rank uint
	if err := row.Scan(&rank); err != nil {
		return err
	}
	rank++

	now := time.Now()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (title, description, priority, duration, deadline, rank, done, created_at, updated_at)
		     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.Title, t.Description, t.Priority, t.Duration, t.Deadline, rank, t.Done, now, now)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	taskID := uint(id)

	if len(t.Tags) > 0 {
		values := make([]string, len(t.Tags))
		params := make([]interface{}, 2*len(t.Tags))
		for i, tag := range t.Tags {
			values[i] = "(?, ?)"
			params[i*2] = taskID
			params[i*2+1] = tag
		}
		_, err := r.db.ExecContext(
			ctx,
			fmt.Sprintf("INSERT INTO tags (task_id, tag) VALUES %s", strings.Join(values, ",")),
			params...,
		)
		if err != nil {
			return err
		}
	}

	if len(t.Dependencies) > 0 {
		values := join("(?, ?, ?)", ",", len(t.Dependencies))
		params := make([]interface{}, 3*len(t.Dependencies))
		for i, dep := range t.Dependencies {
			params[i*3+0] = taskID
			params[i*3+1] = dep.ID
			params[i*3+2] = now
		}

		_, err := r.db.ExecContext(
			ctx,
			fmt.Sprintf(`
				INSERT INTO task_dependencies (task_id, dependency_task_id, created_at)
				VALUES %s`,
				values,
			), params...,
		)
		if err != nil {
			return err
		}
	}

	t.ID = taskID
	t.Rank = rank
	return nil
}

func (r *TaskRepository) Update(ctx context.Context, t *tonight.Task) error {
	if t.ID == 0 {
		return errors.New("cannot insert a task")
	}

	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET
			title = ?,
			description = ?,
			priority = ?,
			duration = ?,
			deadline = ?,
			updated_at = ?
		WHERE id = ?
	`, t.Title, t.Description, t.Priority, t.Duration, t.Deadline, now, t.ID)
	if err != nil {
		return err
	}

	if _, err := r.db.ExecContext(ctx, "DELETE FROM tags WHERE task_id = ?", t.ID); err != nil {
		return err
	}

	if len(t.Tags) > 0 {
		values := make([]string, len(t.Tags))
		params := make([]interface{}, 2*len(t.Tags))
		for i, tag := range t.Tags {
			values[i] = "(?, ?)"
			params[i*2] = t.ID
			params[i*2+1] = tag
		}
		_, err := r.db.ExecContext(
			ctx,
			fmt.Sprintf("INSERT INTO tags (task_id, tag) VALUES %s", strings.Join(values, ",")),
			params...,
		)
		if err != nil {
			return err
		}
	}

	if _, err := r.db.ExecContext(ctx, "DELETE FROM task_dependencies WHERE task_id = ?", t.ID); err != nil {
		return err
	}

	if len(t.Dependencies) > 0 {
		values := join("(?, ?, ?)", ",", len(t.Dependencies))
		params := make([]interface{}, 3*len(t.Dependencies))
		for i, dep := range t.Dependencies {
			params[i*3+0] = t.ID
			params[i*3+1] = dep.ID
			params[i*3+2] = now
		}

		_, err := r.db.ExecContext(
			ctx,
			fmt.Sprintf(`
				INSERT INTO task_dependencies (task_id, dependency_task_id, created_at)
				VALUES %s`,
				values,
			), params...,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRepository) MarkDone(ctx context.Context, taskID uint, log tonight.Log) error {
	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO task_logs (task_id, type, completion, description, created_at)
			VALUES (?, ?, ?, ?, ?)`,
		taskID, string(log.Type), log.Completion, log.Description, now,
	)
	if err != nil {
		return err
	}

	if log.Completion == 100 {
		_, err := r.db.ExecContext(
			ctx,
			"UPDATE tasks SET done = ?, done_at = ?, updated_at = ? WHERE id = ?",
			true, now, now, taskID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, taskID uint) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE tasks SET deleted = ? WHERE id = ?",
		true, taskID,
	)
	return err
}

func (r *TaskRepository) UpdateRanks(ctx context.Context, ranks map[uint]uint) error {
	for id, rank := range ranks {
		_, err := r.db.ExecContext(ctx, "UPDATE tasks SET rank = ? WHERE id = ?", rank, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *TaskRepository) StartPlanning(ctx context.Context, userID uint, duration string, taskIDs []uint) (tonight.Planning, error) {
	now := time.Now()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO planning (user_id, duration, startedAt, dismissed) VALUES (?, ?, ?, ?)
		`, userID, duration, now, false)
	if err != nil {
		return tonight.Planning{}, err
	}

	planningID, err := res.LastInsertId()
	if err != nil {
		return tonight.Planning{}, err
	}

	for rank, taskID := range taskIDs {
		_, err := r.db.ExecContext(
			ctx,
			"INSERT INTO planning_has_task (planning_id, rank, task_id) VALUE (?, ?, ?)",
			planningID, rank, taskID,
		)
		if err != nil {
			return tonight.Planning{}, err
		}
	}

	tasks, err := r.List(ctx, taskIDs)
	if err != nil {
		return tonight.Planning{}, err
	}

	planning := tonight.Planning{
		ID: uint(planningID),

		Duration: duration,

		StartedAt: now,
		Dismissed: false,

		Tasks: tasks,
	}

	return planning, nil
}

func (r *TaskRepository) CurrentPlanning(ctx context.Context, userID uint) (tonight.Planning, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, duration, startedAt, dismissed FROM planning
		WHERE user_id = ?
		ORDER BY startedAt DESC LIMIT 1
		`, userID,
	)

	var id uint
	var duration string
	var startedAt time.Time
	var dismissed bool
	if err := row.Scan(&id, &duration, &startedAt, &dismissed); err != nil {
		if err == sql.ErrNoRows {
			return tonight.Planning{}, nil
		}
		return tonight.Planning{}, err
	}

	if dismissed {
		return tonight.Planning{}, nil
	}

	planning := tonight.Planning{
		ID:        id,
		Duration:  duration,
		StartedAt: startedAt,
		Dismissed: dismissed,
	}

	rows, err := r.db.QueryContext(ctx, "SELECT task_id FROM planning_has_task WHERE planning_id = ?", id)
	if err != nil {
		return tonight.Planning{}, err
	}
	defer rows.Close()

	taskIDs := make([]uint, 0)
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return tonight.Planning{}, err
		}

		taskIDs = append(taskIDs, id)
	}

	if err := rows.Close(); err != nil {
		return tonight.Planning{}, err
	}

	tasks, err := r.List(ctx, taskIDs)
	if err != nil {
		return tonight.Planning{}, err
	}

	planning.Tasks = tasks
	return planning, nil
}

func (r *TaskRepository) DismissPlanning(ctx context.Context, userID uint) error {
	row := r.db.QueryRowContext(
		ctx,
		"SELECT id FROM planning WHERE user_id = ? ORDER BY startedAt DESC LIMIT 1",
		userID,
	)

	var id uint
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	_, err := r.db.ExecContext(ctx, "UPDATE planning SET dismissed = ? WHERE id = ?", true, id)
	return err
}

func (r *TaskRepository) List(ctx context.Context, ids []uint) ([]tonight.Task, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	params := make([]interface{}, len(ids))
	for i, id := range ids {
		params[i] = id
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, title, description, priority, rank, duration, deadline, done, done_at, created_at
		  FROM tasks
		 WHERE id IN (%s)
`, join("?", ",", len(ids))), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idOrder := make(map[uint]int)
	for i, id := range ids {
		idOrder[id] = i
	}

	tasks, err := r.loadTasks(ctx, rows)
	if err != nil {
		return nil, err
	}

	sort.Sort(&keepOrder{
		idOrder: idOrder,
		tasks:   tasks,
	})

	return tasks, nil
}

func (r *TaskRepository) loadTasks(ctx context.Context, rows *sql.Rows) ([]tonight.Task, error) {
	taskMap := make(map[uint]tonight.Task, 0)
	ids := make([]uint, 0)
	for rows.Next() {
		var id uint
		var title string
		var description string
		var priority int
		var rank uint
		var duration string
		var deadline *time.Time
		var done bool
		var doneAt *time.Time
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &description, &priority, &rank, &duration, &deadline, &done, &doneAt, &createdAt); err != nil {
			return nil, err
		}

		task := tonight.Task{
			ID:          id,
			Title:       title,
			Description: description,

			Priority: priority,
			Rank:     rank,

			Duration: duration,
			Deadline: deadline,

			Done:   done,
			DoneAt: doneAt,

			CreatedAt: createdAt,
		}
		taskMap[task.ID] = task
		ids = append(ids, task.ID)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	marks := join("?", ",", len(ids))
	params := make([]interface{}, len(ids))
	for i, id := range ids {
		params[i] = id
	}

	// Fetch tags
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(
		"SELECT task_id, tag FROM tags WHERE task_id IN (%s)",
		marks,
	), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make(map[uint][]string)
	for rows.Next() {
		var taskID uint
		var tag string
		if err := rows.Scan(&taskID, &tag); err != nil {
			return nil, err
		}

		tags[taskID] = append(tags[taskID], tag)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	// Fetch logs
	rows, err = r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT task_id, type, completion, description, created_at
		FROM task_logs
		WHERE task_id IN (%s)
		ORDER BY task_id, created_at DESC
	`, marks,
	), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make(map[uint][]tonight.Log)
	for rows.Next() {
		var taskID uint
		var logType tonight.LogType
		var completion int
		var description string
		var createdAt time.Time
		if err := rows.Scan(&taskID, &logType, &completion, &description, &createdAt); err != nil {
			return nil, err
		}

		logs[taskID] = append(logs[taskID], tonight.Log{
			Type:        logType,
			Completion:  completion,
			Description: description,
			CreatedAt:   createdAt,
		})
	}

	// Fetch dependencies
	rows, err = r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT task_id, dependency_task_id
		FROM task_dependencies
		JOIN tasks ON tasks.id = dependency_task_id
		WHERE task_id IN (%s) AND tasks.deleted = ?
	`, marks,
	), append(params, false)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dependencies := make(map[uint][]uint)
	for rows.Next() {
		var taskID uint
		var dependencyID uint
		if err := rows.Scan(&taskID, &dependencyID); err != nil {
			return nil, err
		}

		dependencies[taskID] = append(dependencies[taskID], dependencyID)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	tasks := make([]tonight.Task, len(ids))
	for i, id := range ids {
		task := taskMap[id]
		task.Tags = tags[task.ID]
		task.Log = logs[task.ID]

		for _, log := range task.Log {
			if log.Completion > task.Completion {
				task.Completion = log.Completion
			}

			if task.Completion == 100 {
				task.Done = true
				task.DoneAt = &log.CreatedAt
				break
			}
		}

		tasks[i] = task
		taskMap[id] = task
	}

	// Wait for all the tasks to be effectively marked done
	for i, task := range tasks {
		for _, dependencyID := range dependencies[task.ID] {
			task.Dependencies = append(task.Dependencies, tonight.Dependency{
				ID:   dependencyID,
				Done: taskMap[dependencyID].Done,
			})
		}

		tasks[i] = task
	}

	return tasks, nil
}

func (r *TaskRepository) All(ctx context.Context) ([]tonight.Task, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT id, title, description, priority, rank, duration, deadline, done, done_at, created_at
		  FROM tasks
		  WHERE deleted = ?
		`,
		false,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.loadTasks(ctx, rows)
}
