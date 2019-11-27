package migration

type revision interface {
	name() string
	version() uint
	up() (string, error)
	down() (string, error)
}
