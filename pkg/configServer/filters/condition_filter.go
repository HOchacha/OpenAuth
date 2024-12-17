package filters

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type ConditionFilter struct {
	FilterName string      `yaml:"filter_name"`
	Conditions []Condition `yaml:"conditions"`
}

type Condition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

func (cf *ConditionFilter) Process(c *gin.Context) (bool, error) {
	log.Infof("start condition filter: %s", cf.FilterName)
	for _, condition := range cf.Conditions {
		if !evaluateCondition(c, condition) {
			return false, fmt.Errorf("condition not met: %v", condition)
		}
	}
	return true, nil
}

func evaluateCondition(c *gin.Context, condition Condition) bool {
	var fieldValue string

	switch condition.Field {
	case "header":
		fieldValue = c.GetHeader(condition.Value.(string))
	case "param":
		fieldValue = c.Param(condition.Value.(string))
	case "query":
		fieldValue = c.Query(condition.Value.(string))
	default:
		return false
	}

	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value.(string)
	case "contains":
		return strings.Contains(fieldValue, condition.Value.(string))
	case "prefix":
		return strings.HasPrefix(fieldValue, condition.Value.(string))
	case "suffix":
		return strings.HasSuffix(fieldValue, condition.Value.(string))
	default:
		return false
	}
}
