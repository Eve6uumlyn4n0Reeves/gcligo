package version

var (
	// Version is set via -ldflags "-X gcli2api-go/internal/version.Version=..."
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
)
