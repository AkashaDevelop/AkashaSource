package claude

// ModelList for Claude models
var ModelList = []string{
	// Claude 4 series
	"claude-opus-4-20250514",
	"claude-sonnet-4-20250514",
	// Claude 3.7
	"claude-3-7-sonnet-20250219",
	"claude-3-7-sonnet-latest",
	// Claude 3.5 series
	"claude-3-5-sonnet-20241022",
	"claude-3-5-sonnet-20240620",
	"claude-3-5-sonnet-latest",
	"claude-3-5-haiku-20241022",
	"claude-3-5-haiku-latest",
	// Claude 3 series
	"claude-3-opus-20240229",
	"claude-3-sonnet-20240229",
	"claude-3-haiku-20240307",
	// Legacy
	"claude-2.1",
	"claude-2.0",
	"claude-instant-1.2",
}

const (
	BaseURL = "https://api.anthropic.com"
)
