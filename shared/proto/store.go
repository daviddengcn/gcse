package gcsepb

func (m *Repository) PutPackage(path string, pkg *Package) {
	if m.Packages == nil {
		m.Packages = make(map[string]*Package)
	}
	m.Packages[path] = pkg
}
