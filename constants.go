package github

// API groups useful info
type API struct {
	URL     string
	STABLE  string
	PREVIEW string
}

// APIURLs provides Github URL info
var APIURLs = API{
	URL:     "https://api.github.com",
	STABLE:  "application/vnd.github.v3+json",
	PREVIEW: "application/vnd.github.korra-preview",
}
