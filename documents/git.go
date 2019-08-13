package documents

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os/exec"
	"path/filepath"
)

// FindGit returns the path to the git executable, or an empty string if it is
// not available.
func FindGit() string {
	git, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	return git
}

// gitRepo returns the path to the repository in which a file belongs.
func gitRepo(git, file string) string {
	b, err := exec.Command(git, "-C", filepath.Dir(file), "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.Clean(string(bytes.TrimRight(b, "\n")))
}

// gitStatus returns the status of a file as known by git. A result of 1 means
// the file is modified, 0 means the file may or may not exist, but is
// untouched, -1 means untracked or ignored, and -2 means an error occurred.
func gitStatus(git, repo, file string) int {
	if git == "" {
		return 0
	}
	b, err := exec.Command(git, "-C", repo, "status", "--porcelain", file).Output()
	if err != nil {
		return -2
	}
	if len(b) < 2 {
		return 0
	}
	if bytes.ContainsAny(b[0:2], "?!") {
		return -1
	}
	return 1
}

// GitRead attempts to read the most recently committed content of a file.
func GitRead(git, file string) (b []byte, err error) {
	if git != "" {
		if repo := gitRepo(git, file); repo != "" {
			// File path relative to repo.
			rel, err := filepath.Rel(repo, file)
			if err != nil {
				goto nofile
			}
			rel = filepath.ToSlash(rel)

			status := gitStatus(git, repo, rel)
			switch status {
			case -1, -2:
				goto nofile
			case 1:
				// Read the file via git-show.
				if b, err = exec.Command(git, "-C", repo, "show", "HEAD:"+rel).Output(); err != nil {
					goto nofile
				}
				return b, nil
			}
		}
	}

	// Try reading the file normally.
	return ioutil.ReadFile(file)

nofile:
	// Pretend the file does not exist.
	return nil, errors.New("file does not exist")
}
