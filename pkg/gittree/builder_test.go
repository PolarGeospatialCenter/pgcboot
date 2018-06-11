package treebuilder

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	git "gopkg.in/src-d/go-git.v4"
)

func TestSSHLoad(t *testing.T) {
	b := &Builder{options: &git.CloneOptions{}}
	sshKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA4oSIRw532DOReuRcgfbQTowqgi+Yvf5r8JkfYAyAI/vmnLAD
Cz+vPeEc7SAlG+hozQQ/HS6d4WoiMqj8+p1DiHRjONKekXwbTGUlX1Kk/dgU9Kmi
isd1KIsHh5+m0LkgQSIe+1/Dh+7zNRVNWwct+XySJACTfq1mJql8i5egsRWsv4Sf
FxX4N3wRmkeAf2sHVm88YO9NTmQ/3HPDxW/jy6BREhf2UrPOQHAc5THGKJSw9HUa
BB+/9O+5HdkPKPcb6AuqUSQ9S4+0F2r6BWsw+npAN3N7BPPp0nRvarvskYdHMwsE
27FQMzr7bzR4CYPY1DeXkNJpc6zsZPqB5LDTnQIDAQABAoIBAQC3pFMHqIco1MYB
J9qH0x2WULS1zvi6L+Y6rSluqTPJ+JNCPMB7AiqEtFjLNeBf+8/bRrIUapK9CVqo
T7CpTY5Otm0qyDaeJEvNZ8MgwNPaqLB0moKYmJQ3Rl/YaGrJlQy9QXh0u3K+Zc7v
HlIUloGDXqbsYTsy3EmQ1p+OXGN+rFlcexKXvqiyCPgnaKofTWSpPczpEsYU2S8L
svF3Hq1eGupr/Mn5fVdZGqxCR/Mc7YQ1zBW3Fq7tcwTed5lNL6zBLWPozPt/ug6u
wULzNHZ0hl3yCEyX8xYAAneOuFyqNROSJml75XQWinKw/J5eSV4B4OJaC2bOqD6s
LViNKXoBAoGBAPRWBYfiqSc/Jin8IzLzFR6scyLzGKQxPEuzFZY9ByWVnTGOgWdL
3skJK7ePciEnsWvVLUndoI/ncCwvWIFjt7YMFhNE1B90hh9stV8oOCNNqWOge775
HzeDntRiZTja6Z2Vo+ouHj1KeKbDONTLqydr3nO0cH6qK11EwQO6QK/9AoGBAO1U
wcJy66rdTqDdmmE04M7Jiqs2rnFDZZG7O9j0PVGznIMtwEZWkRlh90J8N01nkeoK
3odDyhxHOuPJa0wL73bkaOTIqUpHkEQy/MBxlUxEqXTN05Dzfauuts9QV5J+d0gs
78nRVKFVu/vsakQxkiwtvr60DjEzKfVACzcnIfQhAoGBAMF337dCNXhbG2gBOwnb
yqxYFm7lGGzig4DZU817k04iUq7rzPEy9TwwI8qcLd2s5WKiENM9RybLNln2P1ls
0Qm4Nj6ZsHEbvhvh4xdu7Eyf8PFvIK0N67b0ZG59XvMO/A6Ib5s9Wzpi3ngFetmc
T3DOi/0IMk9JhT678y11bEUtAoGBAJO6gGljU2KmIv1rM19ypMTTGyf7/5WtGBog
a95eGZUzsibNYbPmyqb8HgcafuoFoAQJA/86qSH1DKkhhVJu0340Kz7N0OLVrO1m
t4Gqsf4pdzmnrRu7FOy68jwVjI05f1JD9navgHh0f3EO9g7AtHYfe24FchgZ+vIY
DWMlTrNBAoGBAIbyvTJ2WzfRWk3Z7RLs6Cg1QafCJ8ZVadnXrx8+LNbF90U29FQF
B76T517twaSm96Az4UHgwRNQ2z96ZVXvAJQqT8Rcg1d/eId60qYLsUid8F8Disg8
hXS0RKBe1gv5TqVFmDe91FNgXVhLZHWTn3EtLFvVVieEx5OK6nxU5+oV
-----END RSA PRIVATE KEY-----`
	err := b.SetSSHKey(sshKey)
	if err != nil {
		t.Errorf("Error loading SSH Key: %v", err)
	}
}

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

	cwd, _ := os.Getwd()
	testRepoPath := path.Join(cwd, "..", "..", "test", "repo")
	t.Logf("Creating new local git tree builder from repo: %s", path.Clean(testRepoPath))
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
