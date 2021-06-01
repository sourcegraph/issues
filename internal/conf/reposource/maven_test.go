package reposource

import "testing"

func TestDecomposeMavenPath(t *testing.T) {
	obtained := DecomposeMavenPath("//maven/maven/junit:junit:4.13.2")
	if obtained != "junit:junit:4.13.2" {
		t.Fail()
	}
}
