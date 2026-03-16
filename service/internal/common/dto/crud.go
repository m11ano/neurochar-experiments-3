package dto

type UpdateMeta struct {
	Version          int64 `json:"_version,omitempty"`
	SkipVersionCheck bool  `json:"_skipVersionCheck"`
}
