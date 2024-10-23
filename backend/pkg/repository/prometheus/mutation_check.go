package prometheus

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type MutationPQLCheck struct {
	PQL        string
	UpperLimit string
	LowerLimit string
}

func (c *MutationPQLCheck) GetMurationMessage(value float64, labels string) string {
	if c.UpperLimit == "" && c.LowerLimit == "" {
		return "Metric: " + c.PQL +
			"\nValue: " + strconv.FormatFloat(value, 'g', -1, 64) +
			"\nLabels: " + fmt.Sprintf("{%+v}", labels)
	} else {
		return "Metric: " + c.PQL +
			"\nValue: " + strconv.FormatFloat(value, 'g', -1, 64) +
			"\nLimit: " + fmt.Sprintf("{upper: %s, lower: %s}", c.UpperLimit, c.LowerLimit) +
			"\nLabels: " + fmt.Sprintf("{%s}", labels)
	}
}

func (r *promRepo) ExecutedMutationCheck(c *MutationPQLCheck, startTime, endTime time.Time, step time.Duration) ([]MetricResult, error) {
	withComparison := []string{}
	if len(c.UpperLimit) > 0 {
		limit, err := parseMutationLimit(c.PQL, c.UpperLimit)
		if err != nil {
			return nil, err
		}
		withComparison = append(withComparison, fmt.Sprintf("((%s) > (%s))", c.PQL, limit))
	}
	if len(c.LowerLimit) > 0 {
		limit, err := parseMutationLimit(c.PQL, c.LowerLimit)
		if err != nil {
			return nil, err
		}
		withComparison = append(withComparison, fmt.Sprintf("((%s) < (%s))", c.PQL, limit))
	}

	var pql string
	if len(withComparison) > 0 {
		pql = strings.Join(withComparison, " or ")
	} else {
		pql = c.PQL
	}

	return r.QueryRangeData(startTime, endTime, pql, step)
}

func parseMutationLimit(pql string, s string) (string, error) {
	if strings.HasPrefix(s, "raw.") {
		return s[4:], nil
	} else if strings.HasPrefix(s, "avg.") {
		// xxx > avg.1h; xxx > avg.2d
		return fmt.Sprintf("avg_over_time(%s[%s])", pql, s[4:]), nil
	} else if strings.HasPrefix(s, "dod") {
		// xxx > dod * 1.2
		return fmt.Sprintf("((%s) offset 24h ) %s", pql, s[3:]), nil
	} else if strings.HasPrefix(s, "wow") {
		return fmt.Sprintf("((%s) offset 7d ) %s", pql, s[3:]), nil
	} else if strings.HasPrefix(s, "pct.") {
		// 特殊,只适用bucket指标, 先算出静态值,再进行百分比计算
		// xxx >
		// TODO
	}

	return "", fmt.Errorf("invalid limit: %s", s)
}
