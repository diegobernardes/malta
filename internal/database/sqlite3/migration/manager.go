package migration

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/golang-migrate/migrate/v4/source"
)

// Manager is responsible for managing the revisions.
type Manager struct {
	revisions []revision
}

// Init internal state.
func (m *Manager) Init() {
	m.revisions = []revision{revision0{}}
	source.Register("static", m)
}

// Open is not used, it's here just to fulfill the source.Driver interface.
func (m *Manager) Open(url string) (source.Driver, error) {
	return m, nil
}

// Close is not used, it's here to fulfill the source.Driver interface.
func (m *Manager) Close() error {
	return nil
}

// First returns the very first migration version available.
func (m *Manager) First() (version uint, err error) {
	if len(m.revisions) == 0 {
		return 0, os.ErrNotExist
	}
	return m.revisions[0].version(), nil
}

// Prev returns the previous version for a given version available.
func (m *Manager) Prev(version uint) (prevVersion uint, err error) {
	return m.revision((int)(version) - 1)
}

// Next returns the next version for a given version available.
func (m *Manager) Next(version uint) (nextVersion uint, err error) {
	return m.revision((int)(version) + 1)
}

// ReadUp returns the UP revision body and an identifier that helps finding this migration in the
// source for a given version.
func (m *Manager) ReadUp(version uint) (r io.ReadCloser, identifier string, err error) {
	revision := m.revisions[version]
	buf, err := m.payload(revision.up)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate the up revision '%d': %w", version, err)
	}
	return buf, revision.name(), nil
}

// ReadDown returns the DOWN revision body and an identifier that help finding this migration in
// the source for a given version.
func (m *Manager) ReadDown(version uint) (r io.ReadCloser, identifier string, err error) {
	revision := m.revisions[version]
	buf, err := m.payload(revision.down)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate the down revision '%d': %w", version, err)
	}
	return buf, revision.name(), nil
}

func (m *Manager) payload(fn func() (string, error)) (io.ReadCloser, error) {
	payload, err := fn()
	if err != nil {
		return nil, fmt.Errorf("failed to generate the payload: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})
	if _, err := buf.WriteString(payload); err != nil {
		return nil, fmt.Errorf("failed to generate the read closer: %w", err)
	}
	return ioutil.NopCloser(buf), nil
}

func (m *Manager) revision(version int) (uint, error) {
	if (version < 0) || (version >= len(m.revisions)) {
		return 0, os.ErrNotExist
	}
	return m.revisions[version].version(), nil
}
