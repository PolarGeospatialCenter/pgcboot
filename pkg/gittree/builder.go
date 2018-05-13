package treebuilder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

// GetPathFromRef returns the top-level subfolder name of the branch or tag
func GetPathFromRef(ref *plumbing.Reference) string {
	if ref.Type() == plumbing.HashReference && ref.Name().IsRemote() {
		var branch string
		fmt.Sscanf(ref.Name().Short(), "origin/%s", &branch)
		return fmt.Sprintf("branch/%s", branch)
	} else if ref.Type() == plumbing.HashReference && ref.Name().IsTag() {
		tag := ref.Name().Short()
		return fmt.Sprintf("release/%s", tag)
	}
	return ""
}

// Builder creates a file tree containing all branches and tags of a git repo
// with one top-level folder per branch or tag.
type Builder struct {
	options *git.CloneOptions
	store   storage.Storer
	path    string
}

// NewLocalBuilder creates a builder that points directly to a local bare repository.
func NewLocalBuilder(treePath, localRepoPath, bareRepoPath string) (*Builder, error) {
	store, err := filesystem.NewStorage(osfs.New(localRepoPath))
	if err != nil {
		return nil, err
	}

	b := &Builder{path: treePath, store: store, options: &git.CloneOptions{URL: bareRepoPath}}
	return b, nil
}

// NewSSHBuilder creates a builder object pointing to a remote git repo via ssh.
// The repo is cloned (bare) to repo_path.  If repo_path doesn't exist it will be created.
// If repoPath already exists and contains a git repo, any updates will be fetched.
//
// The treePath is the root of the output file tree.
func NewSSHBuilder(remote, deployKey, treePath, repoPath string) (*Builder, error) {
	cloneOptions := &git.CloneOptions{URL: remote}
	b := &Builder{options: cloneOptions, path: treePath}
	err := b.SetSSHKey(deployKey)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(repoPath); os.IsNotExist(err) {
		os.MkdirAll(repoPath, 0755)
	}

	b.store, err = filesystem.NewStorage(osfs.New(repoPath))
	if err != nil {
		return nil, err
	}

	b.fetch()
	return b, nil
}

// SetSSHKey opens the supplied ssh key and configures the builder to use it for cloning and fetching.
func (b *Builder) SetSSHKey(deployKeyPath string) error {
	auth, err := ssh.NewPublicKeys("git", []byte(deployKeyPath), "")
	if err != nil {
		return err
	}
	b.options.Auth = auth
	return nil
}

func (b *Builder) getRepository(worktree billy.Filesystem) (*git.Repository, error) {
	log.Println("Attempting to open existing repo")
	repo, err := git.Open(b.store, worktree)
	if err == git.ErrRepositoryNotExists {
		log.Println("No repo exists, cloning")
		repo, err = git.Clone(b.store, worktree, b.options)
		return repo, err
	}
	return repo, err
}

func (b *Builder) fetch() error {
	repo, err := b.getRepository(nil)
	if err != nil {
		return err
	}

	err = repo.Fetch(&git.FetchOptions{Auth: b.options.Auth})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}

// CloneRef creates a subfolder for a given ref and checks out the current contents.
func (b *Builder) CloneRef(ref *plumbing.Reference) error {
	workpath := filepath.Join(b.path, GetPathFromRef(ref))
	if _, err := os.Stat(workpath); os.IsNotExist(err) {
		os.MkdirAll(workpath, 0755)
	}

	repo, err := b.getRepository(osfs.New(workpath))
	if err != nil {
		return err
	}

	log.Println("Getting worktree")
	tree, err := repo.Worktree()
	if err != nil {
		return err
	}
	hash := ref.Hash()
	if ref.Name().IsTag() {
		tag, err := repo.TagObject(ref.Hash())
		if err == nil {
			commit, err := tag.Commit()
			if err != nil {
				return err
			}
			hash = commit.Hash
		} else if err != plumbing.ErrObjectNotFound {
			return err
		}
	}

	log.Printf("Checking out: %s", hash)
	err = tree.Checkout(&git.CheckoutOptions{Hash: hash, Force: true})
	if err != nil {
		return err
	}
	return nil
}

// FindRefs returns all tag/branch references for the repo
func (b *Builder) FindRefs() ([]*plumbing.Reference, error) {
	log.Println("Clone git repo")
	r, err := b.getRepository(nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	refs, err := r.References()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	references := make([]*plumbing.Reference, 0)

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference && (ref.Name().IsRemote() || ref.Name().IsTag()) {
			references = append(references, ref)
		}
		return nil
	})

	return references, err
}

// BuildGitTree checks out each branch or tag of the repo into the tree_path
func (b *Builder) BuildGitTree() error {
	log.Println("Find all relevant references")
	refs, err := b.FindRefs()
	if err != nil {
		return err
	}
	for _, ref := range refs {
		log.Println(ref.Name().Short())
		log.Println(GetPathFromRef(ref))
		err = b.CloneRef(ref)
		if err != nil {
			return err
		}
	}
	log.Println("Remove now irellevant trees")
	log.Println("Checkout new trees")
	log.Println("Update existing trees")
	return nil
}
