package treebuilder

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

var testRepoPath = "../../test/repo"

func TestLocalBuilder(t *testing.T) {
	repoDir, err := ioutil.TempDir("", "example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(repoDir) // clean up

	workDir, err := ioutil.TempDir("", "example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(workDir) // clean up

	b, err := NewLocalBuilder(workDir, repoDir, path.Clean(testRepoPath))
	if err != nil {
		t.Fatalf("Error creating local builder: %v", err)
	}
	if b.path != workDir {
		t.Fatalf("The output path isn't set properly")
	}

	repo, err := b.getRepository(nil)
	if err != nil {
		t.Fatalf("Unable to open repository: %v", err)
	}

	if repo == nil {
		t.Fatalf("Nil repository returned.")
	}

	err = b.BuildGitTree()
	if err != nil {
		t.Fatalf("Unable to build git tree.")
	}
}
