package domains

type Paging struct {
	Page      int64 `json:"page,omitempty"`
	PageCount int64 `json:"pagecount,omitempty"`
	PageSize  int64 `json:"pagesize,omitempty"`
}
