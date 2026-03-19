package data

type APIResponse[T any] struct {
	Data   T       `json:"data"`
	Paging *Paging `json:"paging,omitempty"`
	Error  *string `json:"error,omitempty"`
}

type Paging struct {
}
