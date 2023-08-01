package ports

type Message struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (m Message) WithError(err error) Message {
	m.Error = err.Error()
	return m
}

func (m Message) WithMessage(message string) Message {
	m.Message = message
	return m
}

type CreateDatabaseRequest struct {
	DatabaseType       string `json:"database_type"`
	MigrationFilesPath string `json:"migration_files_path,omitempty"`
	FixtureFilePath    string `json:"fixture_file_path,omitempty"`
}

type CreateDatabaseResponse struct {
	DatabaseName  string `json:"database_name"`
	ConnectionURI string `json:"connection_uri"`
}
