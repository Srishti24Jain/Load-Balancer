package domain

type RegisterUrls struct {
	Backends []Url `json:"backends"`
}

type Url struct {
	URL string `json:"url"`
}
