// Package blueprints provides blueprint service implementation for gRPC layer.
package blueprints

// refResponse represents the response from getting a git reference.
type refResponse struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

// createRefRequest represents a request to create a git reference.
type createRefRequest struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

// committer represents the committer information for a commit.
type committer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// createFileRequest represents a request to create or update a file.
type createFileRequest struct {
	Message   string    `json:"message"`
	Content   string    `json:"content"` // Already base64-encoded string (API expects base64)
	Branch    string    `json:"branch"`
	Committer committer `json:"committer"`
}

// createFileResponse represents the response from creating a file.
type createFileResponse struct {
	Content struct {
		Name string `json:"name"`
		Path string `json:"path"`
		SHA  string `json:"sha"`
	} `json:"content"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// createPRRequest represents a request to create a pull request.
type createPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// createPRResponse represents the response from creating a pull request.
type createPRResponse struct {
	ID      int    `json:"id"`
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
}
