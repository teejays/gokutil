package filter

import (
	"fmt"
	"strconv"

	"github.com/doug-martin/goqu/v9"
	"github.com/huandu/go-sqlbuilder"
)

type Operator int8

const (
	_ Operator = iota
	EQUAL
	NOT_EQUAL
	IN
	GREATER_THAN
	GREATER_THAN_EQUAL
	LESS_THAN
	LESS_THAN_EQUAL
	LIKE
	ILIKE
	NOT_LIKE
	IS_NULL
	IS_NOT_NULL
)

func (o *Operator) UnmarshalJSON(data []byte) error {
	str, err := strconv.Unquote(string(data))
	if err != nil {
		// Ignore error because that usually means there are no qoutes, or str is too small
		str = string(data) // since with error, Unquote returns an empty string
	}
	return o.FromString(str)
}

func (f Operator) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, f.String()), nil
}

func (o Operator) String() string {
	switch o {
	case EQUAL:
		return "EQUAL"
	case NOT_EQUAL:
		return "NOT_EQUAL"
	case GREATER_THAN:
		return "GREATER_THAN"
	case GREATER_THAN_EQUAL:
		return "GREATER_THAN_EQUAL"
	case LESS_THAN:
		return "LESS_THAN"
	case LESS_THAN_EQUAL:
		return "LESS_THAN_EQUAL"
	case LIKE:
		return "LIKE"
	case ILIKE:
		return "ILIKE"
	case NOT_LIKE:
		return "NOT_LIKE"
	case IS_NULL:
		return "IS_NULL"
	case IS_NOT_NULL:
		return "IS_NOT_NULL"
	default:
		return "INVALID_OPERATOR"
	}
}

func (o *Operator) FromString(str string) error {
	switch str {
	case "EQUAL":
		*o = EQUAL
	case "NOT_EQUAL":
		*o = NOT_EQUAL
	case "GREATER_THAN":
		*o = GREATER_THAN
	case "GREATER_THAN_EQUAL":
		*o = GREATER_THAN_EQUAL
	case "LESS_THAN":
		*o = LESS_THAN
	case "LESS_THAN_EQUAL":
		*o = LESS_THAN_EQUAL
	case "LIKE":
		*o = LIKE
	case "ILIKE":
		*o = ILIKE
	case "NOT_LIKE":
		*o = NOT_LIKE
	case "IS_NULL":
		*o = IS_NULL
	case "IS_NOT_NULL":
		*o = IS_NOT_NULL
	default:
		return fmt.Errorf("Unrecognized Operator '%s'", str)
	}
	return nil
}

func (f Operator) GetOperator() Operator { return f }

// ImplementsGraphQLType maps this custom Go type to the graphql scalar type in the schema.
func (f Operator) ImplementsGraphQLType(name string) bool {
	return name == "FilterOperator"
}

func (f *Operator) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err := f.FromString(input)
		if err != nil {
			return err
		}
	default:
		err = fmt.Errorf("wrong type for Operator: %T", input)
	}
	return err
}

func IsValidOperator(op Operator) bool { return op > 0 }

type OperatorInfo struct {
	Name string

	Sign           string
	GolangSign     string
	SqlSign        string
	TypescriptSign string

	InjectSqlBuilderWhereCond_Huandu func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error
	InjectSqlBuilderWhereCond_Goqu   func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset

	NoValues            bool
	AllowMultipleValues bool
	MinNumValues        int
	MaxNumValues        int
}

func (t OperatorInfo) String() string {
	return t.Name
}
func (t OperatorInfo) GetSqlSign() string {
	if t.SqlSign != "" {
		return t.SqlSign
	}
	return t.Sign
}
func (t OperatorInfo) GetSqlFormatString() string {
	if t.SqlSign != "" {
		return t.SqlSign
	}
	return t.Sign
}

// func joinStrTimesN(s, sep string, n int) string {
// 	var strs = make([]string, 0, n)
// 	for i := 0; i < n; i++ {
// 		strs = append(strs, s)
// 	}
// 	return strings.Join(strs, sep)
// }

var store = map[Operator]OperatorInfo{
	EQUAL: OperatorInfo{
		Name:    "EQUAL",
		Sign:    "==",
		SqlSign: "=",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.Equal(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Eq(values[0]))
		},
	},
	NOT_EQUAL: OperatorInfo{
		Name: "NOT_EQUAL",
		Sign: "!=",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.NotEqual(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Neq(values[0]))
		},
	},
	IN: OperatorInfo{
		Name:    "IN",
		SqlSign: "IN",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.In(col, values...))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).In(values))
		},
	},
	GREATER_THAN: OperatorInfo{
		Name: "GREATER_THAN",
		Sign: ">",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.GreaterThan(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Gt(values[0]))
		},
	},
	GREATER_THAN_EQUAL: OperatorInfo{
		Name: "GREATER_THAN_EQUAL",
		Sign: ">=",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.GreaterEqualThan(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Gte(values[0]))
		},
	},
	LESS_THAN: OperatorInfo{
		Name: "LESS_THAN",
		Sign: "<",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.LessThan(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Lt(values[0]))
		},
	},
	LESS_THAN_EQUAL: OperatorInfo{
		Name: "LESS_THAN_EQUAL",
		Sign: "<=",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.LessEqualThan(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Lte(values[0]))
		},
	},
	LIKE: OperatorInfo{
		Name:    "LIKE",
		Sign:    "",
		SqlSign: "LIKE",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.Like(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).Like(fmt.Sprintf("%%%s%%", values[0]))) // `%%` results in `%`
		},
	},
	ILIKE: OperatorInfo{
		Name:    "ILIKE",
		Sign:    "",
		SqlSign: "ILIKE",
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.Like(col, values[0])) // No ILike implemented?
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).ILike(fmt.Sprintf("%%%s%%", values[0]))) // `%%` results in `%`
		},
	},
	NOT_LIKE: OperatorInfo{
		Name:                "NOT_LIKE",
		SqlSign:             "NOT LIKE",
		AllowMultipleValues: true,
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.NotLike(col, values[0]))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).NotLike(values[0]))
		},
	},
	IS_NULL: OperatorInfo{
		Name:     "IS_NULL",
		Sign:     "!=",
		SqlSign:  "IS NULL",
		NoValues: true,
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.IsNull(col))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).IsNull())
		},
	},
	IS_NOT_NULL: OperatorInfo{
		Name:     "IS_NOT_NULL",
		Sign:     "",
		SqlSign:  "IS NOT NULL",
		NoValues: true,
		InjectSqlBuilderWhereCond_Huandu: func(sb *sqlbuilder.SelectBuilder, col string, values ...interface{}) error {
			sb.Where(sb.IsNotNull(col))
			return nil
		},
		InjectSqlBuilderWhereCond_Goqu: func(sb *goqu.SelectDataset, col string, values ...interface{}) *goqu.SelectDataset {
			return sb.Where(goqu.C(col).IsNotNull())
		},
	},
}

// func singleSqlCondition(col, sign, d string, values ...interface{}) string {
// 	return fmt.Sprintf("%s %s ?", col, sign)
// }
// func multipleSqlCondition_AND(col, sign, d string, values ...interface{}) string {
// 	singleCond := fmt.Sprint("%s %s %d", col, sign, "?")
// 	multipleConds := joinStrTimesN(singleCond, " AND", len(values))
// 	return multipleConds
// }
// func multipleSqlCondition_OR(col, sign, d string, values ...interface{}) string {
// 	singleCond := fmt.Sprint("%s %s %d", col, sign, "?")
// 	multipleConds := joinStrTimesN(singleCond, " OR", len(values))
// 	return multipleConds
// }

func getOperatorInfo(op Operator) (OperatorInfo, error) {
	if info, exists := store[op]; exists {
		return info, nil
	}
	return OperatorInfo{}, fmt.Errorf("invalid operator: %d", op)
}
