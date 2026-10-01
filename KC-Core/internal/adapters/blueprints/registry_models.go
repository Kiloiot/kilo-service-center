// Package blueprints provides blueprint service implementation for gRPC layer.
package blueprints

// refResponse represents the response from getting a git reference.
type refResponse struct {
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

// CreateFileRequest represents a request to create or update a file.
type CreateFileRequest struct {
	Message   string    `json:"message"`
	Content   string    `json:"content"` // Already base64-encoded string (API expects base64)
	Branch    string    `json:"branch"`
	Committer committer `json:"committer"`
}

// createFileResponse represents the response from creating a file.
type createFileResponse struct {
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// CreatePRRequest represents a request to create a pull request.
type CreatePRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// createPRResponse represents the response from creating a pull request.
type createPRResponse struct {
	HTMLURL string `json:"html_url"`
}
