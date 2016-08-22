package stpb

import "github.com/daviddengcn/gcse/proto/spider"

func (m *Repository) PutPackage(path string, pkg *sppb.Package) {
	if m.Packages == nil {
		m.Packages = make(map[string]*sppb.Package)
	}
	m.Packages[path] = pkg
}
