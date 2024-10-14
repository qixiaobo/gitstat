package internal

import (
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"os"
	"path/filepath"
	"time"
)

type Project struct {
	Name    string  `json:"name"`
	Commits Commits `json:"commits"`

	filepath string
}

func NewProject(path string) (*Project, error) {
	name, err := projectName(path)
	if err != nil {
		return nil, err
	}

	return &Project{
		Name:     name,
		Commits:  make(Commits, 0),
		filepath: path,
	}, nil
}

func (p *Project) ParseCommits() error {
	// Instantiate a new repository targeting the given path (the .git folder)
	fs := osfs.New(p.filepath)
	if _, err := fs.Stat(git.GitDirName); err == nil {
		fs, err = fs.Chroot(git.GitDirName)
		if err != nil {
			return err
		}
	}

	s := filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})
	defer s.Close()

	r, err := git.Open(s, fs)
	if err != nil {
		return err
	}

	// ... retrieve the branch pointed by HEAD
	ref, err := r.Head()
	if err != nil {
		return err
	}

	// ... retrieve the commit history
	// 考虑增加一个环境变量支持since参数 以减少耗时 但是如果没有该环境变量需要忽略改参数
	sinceStr := os.Getenv("GIT_SINCE")
	var options *git.LogOptions
	if sinceStr != "" {
		layout := "2006-01-02"
		sinceTime, err := time.Parse(layout, sinceStr)
		if err != nil {
			return err
		}

		// 使用 since 参数
		options = &git.LogOptions{
			From:  ref.Hash(),
			Since: &sinceTime,
		}
	} else {
		// 如果环境变量不存在，则不使用 since 参数
		options = &git.LogOptions{
			From: ref.Hash(),
		}
	}

	cIter, err := r.Log(options)
	if err != nil {
		return err
	}

	// ... iterate over the commits
	err = cIter.ForEach(func(c *object.Commit) error {
		p.Commits = append(p.Commits, NewCommit(c))
		return nil
	})
	if err != nil {
		return err
	}

	return p.Commits.ParseFileChanges()
}

func projectName(fp string) (string, error) {
	abs, err := filepath.Abs(fp)
	if err != nil {
		return "", err
	}
	for i := len(abs) - 1; i >= 0; i-- {
		if os.IsPathSeparator(abs[i]) {
			return abs[i+1:], nil
		}
	}
	return abs, nil
}
