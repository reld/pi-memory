package projects

type Registry struct {
	Version        int             `json:"version"`
	BaseStorageDir string          `json:"baseStorageDir"`
	Projects       []RegistryEntry `json:"projects"`
}

type RegistryEntry struct {
	ProjectID   string `json:"projectId"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Hash        string `json:"hash"`
	ProjectPath string `json:"projectPath"`
	ProjectDir  string `json:"projectDir"`
	ProjectFile string `json:"projectFile"`
	DBPath      string `json:"dbPath"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type ProjectMetadata struct {
	Version              int      `json:"version"`
	ProjectID            string   `json:"projectId"`
	Name                 string   `json:"name"`
	Slug                 string   `json:"slug"`
	Hash                 string   `json:"hash"`
	ProjectPath          string   `json:"projectPath"`
	ProjectRootStrategy  string   `json:"projectRootStrategy"`
	ProjectDir           string   `json:"projectDir"`
	DBPath               string   `json:"dbPath"`
	CreatedAt            string   `json:"createdAt"`
	UpdatedAt            string   `json:"updatedAt"`
	LastOpenedAt         string   `json:"lastOpenedAt,omitempty"`
	Status               string   `json:"status"`
	PreviousProjectPaths []string `json:"previousProjectPaths,omitempty"`
	RelinkedAt           string   `json:"relinkedAt,omitempty"`
}
