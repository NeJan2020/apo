package clickhouse

import (
	"fmt"

	"github.com/CloudDetail/apo/backend/pkg/model/request"
)

func groupBySQL(req *request.LogIndexRequest) string {
	condition := fmt.Sprintf("timestamp >= toDateTime64(%d, 3) AND timestamp < toDateTime64(%d, 3) AND %s", req.StartTime/1000000, req.EndTime/1000000, req.Query)
	sql := fmt.Sprintf("SELECT count(*) as count, `%s` as f FROM %s.%s WHERE %s GROUP BY %s ORDER BY count DESC LIMIT 10",
		req.Cloumn,
		req.DataBase,
		req.TableName,
		condition,
		req.Cloumn,
	)
	return sql
}

func countSQL(req *request.LogIndexRequest) string {
	condition := fmt.Sprintf("timestamp >= toDateTime64(%d, 3) AND timestamp < toDateTime64(%d, 3) AND %s", req.StartTime/1000000, req.EndTime/1000000, req.Query)
	sql := fmt.Sprintf("SELECT count(*) as count FROM %s.%s WHERE %s",
		req.DataBase,
		req.TableName,
		condition,
	)
	return sql
}

func (ch *chRepo) GetLogIndex(req *request.LogIndexRequest) (map[string]uint64, uint64, error) {
	groupSQL := groupBySQL(req)
	groupRows, err := ch.queryRowsData(groupSQL)
	if err != nil {
		return nil, 0, err
	}
	res := make(map[string]uint64, 0)
	for _, v := range groupRows {
		if v["count"] != nil {
			var key string
			switch v["f"].(type) {
			case string:
				key = v["f"].(string)
			case *string:
				key = *(v["f"].(*string))
			case int16:
				key = fmt.Sprintf("%d", v["f"].(int16))
			case *int16:
				key = fmt.Sprintf("%d", v["f"].(*int16))
			case uint16:
				key = fmt.Sprintf("%d", v["f"].(uint16))
			case int32:
				key = fmt.Sprintf("%d", v["f"].(int32))
			case *int64:
				key = fmt.Sprintf("%d", *(v["f"].(*int64)))
			case int64:
				key = fmt.Sprintf("%d", v["f"].(int64))
			case *float64:
				key = fmt.Sprintf("%f", *(v["f"].(*float64)))
			case float64:
				key = fmt.Sprintf("%f", v["f"].(float64))
			default:
				continue
			}
			if key == "" {
				continue
			}
			res[key] = v["count"].(uint64)
		}
	}
	countSQL := countSQL(req)
	countRows, err := ch.queryRowsData(countSQL)
	if err != nil {
		return nil, 0, err
	}
	if len(countRows) > 0 {
		if countRows[0]["count"] != nil {
			switch countRows[0]["count"].(type) {
			case uint64:
				return res, countRows[0]["count"].(uint64), nil
			}
		}
	}
	return res, 0, nil
}