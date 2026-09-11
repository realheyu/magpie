package configapi

import "time"

type configMetaResp struct {
	AppName   string    `json:"appName"`
	Format    string    `json:"format"`
	Content   string    `json:"content"`
	Sensitive bool      `json:"sensitive"`
	Version   int64     `json:"version"`
	ETag      string    `json:"etag"`
	UpdatedAt time.Time `json:"updatedAt"`
}
