package store

func resetMocks() {
	mocks = dbMocks{}
}

type dbMocks struct {
	extensions mockExtensions
	releases   mockReleases
}

var mocks dbMocks
