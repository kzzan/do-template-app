package v1

type MetaData struct {
	App   string `json:"app"`
	Stage string `json:"stage"`
	Entry string `json:"entry"`
	Title string `json:"title"`
}

type HealthData struct {
	App    string `json:"app"`
	Status string `json:"status"`
}

type ManifestFeature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Route       string `json:"route"`
}

type ManifestData struct {
	App      string            `json:"app"`
	Stage    string            `json:"stage"`
	Entry    string            `json:"entry"`
	Title    string            `json:"title"`
	Headline string            `json:"headline"`
	Features []ManifestFeature `json:"features"`
}
