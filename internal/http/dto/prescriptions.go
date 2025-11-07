package dto

// HistoryQuery binds ?start=&end=&limit=&offset=
type HistoryQuery struct {
	Start  string `form:"start"` // optional: 2025-01-01 or RFC3339
	End    string `form:"end"`   // optional
	Limit  int    `form:"limit,default=50"`
	Offset int    `form:"offset,default=0"`
}
