package migration

type revision0 struct{}

func (revision0) name() string {
	return "Revision 0"
}

func (revision0) version() uint {
	return 0
}

func (revision0) up() (string, error) {
	return `
		CREATE TABLE node (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			address    TEXT NOT NULL,
			metadata   JSON,
			created_at DATETIME NOT NULL
		)
	`, nil
}

func (revision0) down() (string, error) {
	return "DROP TABLE node", nil
}
