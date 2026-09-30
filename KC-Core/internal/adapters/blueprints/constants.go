package blueprints

// HTTP header names and media types specific to the registry HTTP client.
// Shared header vocabulary (Authorization, Content-Type, JSON media type)
// comes from KC-Core/pkg/blueprint.
const (
	// headerAccept is the standard Accept header name.
	headerAccept = "Accept"
	// mediaTypeGitHubJSON is the GitHub REST API media type.
	mediaTypeGitHubJSON = "application/vnd.github+json"
)
