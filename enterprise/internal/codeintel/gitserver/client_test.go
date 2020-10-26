package gitserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseParents(t *testing.T) {
	commits := []string{
		"9ad62c7ec68e377b41a8b8dd846e573b76634172 1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3 683cafd122632142bda6e36563f5719e5b0fa37d",
		"683cafd122632142bda6e36563f5719e5b0fa37d 1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3",
		"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3 02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d",
		"02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d a94fb112d1f2e70f55851c6c569916a9e31caee1",
		"a94fb112d1f2e70f55851c6c569916a9e31caee1 2716762a5213f5fe2576d2a52d1182282704004c",
		"2716762a5213f5fe2576d2a52d1182282704004c 7696eb70f56308b7286cf1fb156af6bbb57495e4",
		"7696eb70f56308b7286cf1fb156af6bbb57495e4 33d2968a49e1131825a0accdd64bdf9901cf144c",
		"33d2968a49e1131825a0accdd64bdf9901cf144c 0b3eb4b7b8da63598d99905dd63e1c9ba92aaf22",
		"0b3eb4b7b8da63598d99905dd63e1c9ba92aaf22 843793335d0f8d70c1d9f1c948ff584a979b5bab",
		"843793335d0f8d70c1d9f1c948ff584a979b5bab 61b1debf13ba14e09e29042727d698073bb87f83 5f93bf0eb32807657bc4dafcc934f817991bd805",
		"5f93bf0eb32807657bc4dafcc934f817991bd805 189f2dd3ec0761e60d054f1dc880cfbf074ede16",
		"189f2dd3ec0761e60d054f1dc880cfbf074ede16 61b1debf13ba14e09e29042727d698073bb87f83",
		"61b1debf13ba14e09e29042727d698073bb87f83 f9fdcbb3871320e0655530b707a3d276d561311a",
		"f9fdcbb3871320e0655530b707a3d276d561311a 6d238ad929833c066db4fb3305e4614c212efb42",
		"6d238ad929833c066db4fb3305e4614c212efb42 c855987d88ef32861b61bfa1118781e63a7b0457",
		"c855987d88ef32861b61bfa1118781e63a7b0457 267fa34a98ba30e53d228500831b6db37883c199",
		"267fa34a98ba30e53d228500831b6db37883c199 7b8acaa20dd8db659a31520d95ae74afe0e43499",
		"7b8acaa20dd8db659a31520d95ae74afe0e43499 e883e8a1bf875f6ed5e095e4355acc54ef38ea2c",
		"e883e8a1bf875f6ed5e095e4355acc54ef38ea2c 8c25906a7dbc904acc96b5435ec42fa5da8b232c",
		"8c25906a7dbc904acc96b5435ec42fa5da8b232c",
	}

	expected := map[string][]string{
		"9ad62c7ec68e377b41a8b8dd846e573b76634172": {"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3", "683cafd122632142bda6e36563f5719e5b0fa37d"},
		"683cafd122632142bda6e36563f5719e5b0fa37d": {"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3"},
		"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3": {"02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d"},
		"02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d": {"a94fb112d1f2e70f55851c6c569916a9e31caee1"},
		"a94fb112d1f2e70f55851c6c569916a9e31caee1": {"2716762a5213f5fe2576d2a52d1182282704004c"},
		"2716762a5213f5fe2576d2a52d1182282704004c": {"7696eb70f56308b7286cf1fb156af6bbb57495e4"},
		"7696eb70f56308b7286cf1fb156af6bbb57495e4": {"33d2968a49e1131825a0accdd64bdf9901cf144c"},
		"33d2968a49e1131825a0accdd64bdf9901cf144c": {"0b3eb4b7b8da63598d99905dd63e1c9ba92aaf22"},
		"0b3eb4b7b8da63598d99905dd63e1c9ba92aaf22": {"843793335d0f8d70c1d9f1c948ff584a979b5bab"},
		"843793335d0f8d70c1d9f1c948ff584a979b5bab": {"61b1debf13ba14e09e29042727d698073bb87f83", "5f93bf0eb32807657bc4dafcc934f817991bd805"},
		"5f93bf0eb32807657bc4dafcc934f817991bd805": {"189f2dd3ec0761e60d054f1dc880cfbf074ede16"},
		"189f2dd3ec0761e60d054f1dc880cfbf074ede16": {"61b1debf13ba14e09e29042727d698073bb87f83"},
		"61b1debf13ba14e09e29042727d698073bb87f83": {"f9fdcbb3871320e0655530b707a3d276d561311a"},
		"f9fdcbb3871320e0655530b707a3d276d561311a": {"6d238ad929833c066db4fb3305e4614c212efb42"},
		"6d238ad929833c066db4fb3305e4614c212efb42": {"c855987d88ef32861b61bfa1118781e63a7b0457"},
		"c855987d88ef32861b61bfa1118781e63a7b0457": {"267fa34a98ba30e53d228500831b6db37883c199"},
		"267fa34a98ba30e53d228500831b6db37883c199": {"7b8acaa20dd8db659a31520d95ae74afe0e43499"},
		"7b8acaa20dd8db659a31520d95ae74afe0e43499": {"e883e8a1bf875f6ed5e095e4355acc54ef38ea2c"},
		"e883e8a1bf875f6ed5e095e4355acc54ef38ea2c": {"8c25906a7dbc904acc96b5435ec42fa5da8b232c"},
		"8c25906a7dbc904acc96b5435ec42fa5da8b232c": {},
	}

	if diff := cmp.Diff(expected, parseParents(commits)); diff != "" {
		t.Errorf("unexpected commit mapping (-want +got):\n%s", diff)
	}
}

func TestParseParentsPartial(t *testing.T) {
	commits := []string{
		"9ad62c7ec68e377b41a8b8dd846e573b76634172 1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3 683cafd122632142bda6e36563f5719e5b0fa37d",
		"683cafd122632142bda6e36563f5719e5b0fa37d 1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3",
		"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3 02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d",
		"a94fb112d1f2e70f55851c6c569916a9e31caee1 2716762a5213f5fe2576d2a52d1182282704004c",
	}

	expected := map[string][]string{
		"9ad62c7ec68e377b41a8b8dd846e573b76634172": {"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3", "683cafd122632142bda6e36563f5719e5b0fa37d"},
		"683cafd122632142bda6e36563f5719e5b0fa37d": {"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3"},
		"1afa9c06d8bb8b2c5746e539ed4eb80c23b21db3": {"02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d"},
		"a94fb112d1f2e70f55851c6c569916a9e31caee1": {"2716762a5213f5fe2576d2a52d1182282704004c"},
		"02f41985f46b400b7a673c3dfb6bab8fd1ac6a6d": {},
		"2716762a5213f5fe2576d2a52d1182282704004c": {},
	}

	if diff := cmp.Diff(expected, parseParents(commits)); diff != "" {
		t.Errorf("unexpected commit mapping (-want +got):\n%s", diff)
	}
}

func TestParseDirectoryChildrenRoot(t *testing.T) {
	dirnames := []string{""}
	paths := []string{
		".github",
		".gitignore",
		"LICENSE",
		"README.md",
		"cmd",
		"go.mod",
		"go.sum",
		"internal",
		"protocol",
	}

	expected := map[string][]string{
		"": paths,
	}

	if diff := cmp.Diff(expected, parseDirectoryChildren(dirnames, paths)); diff != "" {
		t.Errorf("unexpected directory children result (-want +got):\n%s", diff)
	}
}

func TestParseDirectoryChildrenNonRoot(t *testing.T) {
	dirnames := []string{"cmd/", "protocol/", "cmd/protocol/"}
	paths := []string{
		"cmd/lsif-go",
		"protocol/protocol.go",
		"protocol/writer.go",
	}

	expected := map[string][]string{
		"cmd/":          {"cmd/lsif-go"},
		"protocol/":     {"protocol/protocol.go", "protocol/writer.go"},
		"cmd/protocol/": nil,
	}

	if diff := cmp.Diff(expected, parseDirectoryChildren(dirnames, paths)); diff != "" {
		t.Errorf("unexpected directory children result (-want +got):\n%s", diff)
	}
}

func TestParseDirectoryChildrenDifferentDepths(t *testing.T) {
	dirnames := []string{"cmd/", "protocol/", "cmd/protocol/"}
	paths := []string{
		"cmd/lsif-go",
		"protocol/protocol.go",
		"protocol/writer.go",
		"cmd/protocol/main.go",
	}

	expected := map[string][]string{
		"cmd/":          {"cmd/lsif-go"},
		"protocol/":     {"protocol/protocol.go", "protocol/writer.go"},
		"cmd/protocol/": {"cmd/protocol/main.go"},
	}

	if diff := cmp.Diff(expected, parseDirectoryChildren(dirnames, paths)); diff != "" {
		t.Errorf("unexpected directory children result (-want +got):\n%s", diff)
	}
}

func TestCleanDirectoriesForLsTree(t *testing.T) {
	args := []string{"", "foo", "bar/", "baz"}
	actual := cleanDirectoriesForLsTree(args)
	expected := []string{".", "foo/", "bar/", "baz/"}

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected ls-tree args (-want +got):\n%s", diff)
	}
}
