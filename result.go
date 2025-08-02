package morm

type Result struct {
	Error        error
	RowsAffected int64
}

func new_result(e error, rows int64) Result {
	return Result{e, rows}
}

func error_result(e error) Result {
	return Result{Error: e}
}
