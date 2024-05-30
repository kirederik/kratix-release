package writers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-logr/logr"
	"github.com/syntasso/kratix/api/v1alpha1"
)

type GitWriter struct {
	Log       logr.Logger
	gitServer gitServer
	author    gitAuthor
	path      string
}

type gitServer struct {
	URL    string
	Branch string
	Auth   transport.AuthMethod
}

type gitAuthor struct {
	Name  string
	Email string
}

func NewGitWriter(logger logr.Logger, stateStoreSpec v1alpha1.GitStateStoreSpec, destination v1alpha1.Destination, creds map[string][]byte) (StateStoreWriter, error) {
	var authMethod transport.AuthMethod
	switch stateStoreSpec.AuthMethod {
	case v1alpha1.SSHAuthMethod:
		sshKey, ok := creds["sshPrivateKey"]
		if !ok {
			return nil, fmt.Errorf("sshPrivateKey not found in secret %s/%s", destination.Namespace, stateStoreSpec.SecretRef.Name)
		}

		knownHosts, ok := creds["knownHosts"]
		if !ok {
			return nil, fmt.Errorf("knownHosts not found in secret %s/%s", destination.Namespace, stateStoreSpec.SecretRef.Name)
		}

		sshPrivateKey, err := ssh.NewPublicKeys("git", sshKey, "")
		if err != nil {
			return nil, fmt.Errorf("error parsing sshPrivateKey: %w", err)
		}

		knownHostsFile, err := os.CreateTemp("", "knownHosts")
		if err != nil {
			return nil, fmt.Errorf("error creating knownhosts file: %w", err)
		}

		knownHostsFile.Write(knownHosts)
		knownHostsCallback, err := ssh.NewKnownHostsCallback(knownHostsFile.Name())
		if err != nil {
			return nil, fmt.Errorf("error parsing known hosts: %w", err)
		}

		sshPrivateKey.HostKeyCallback = knownHostsCallback
		err = os.Remove(knownHostsFile.Name())
		if err != nil {
			return nil, fmt.Errorf("error removing knownhosts file: %w", err)
		}

		authMethod = sshPrivateKey
	case v1alpha1.BasicAuthMethod:
		username, ok := creds["username"]
		if !ok {
			return nil, fmt.Errorf("username not found in secret %s/%s", destination.Namespace, stateStoreSpec.SecretRef.Name)
		}

		password, ok := creds["password"]
		if !ok {
			return nil, fmt.Errorf("password not found in secret %s/%s", destination.Namespace, stateStoreSpec.SecretRef.Name)
		}

		authMethod = &http.BasicAuth{
			Username: string(username),
			Password: string(password),
		}
	}

	return &GitWriter{
		gitServer: gitServer{
			URL:    stateStoreSpec.URL,
			Branch: stateStoreSpec.Branch,
			Auth:   authMethod,
		},
		author: gitAuthor{
			Name:  "Kratix",
			Email: "kratix@syntasso.io",
		},
		Log:  logger,
		path: filepath.Join(stateStoreSpec.Path, destination.Spec.Path, destination.Namespace, destination.Name),
	}, nil
}

func (g *GitWriter) UpdateFiles(subDir string, workPlacementName string, workloadsToCreate []v1alpha1.Workload, workloadsToDelete []string) error {
	return g.update(subDir, workPlacementName, workloadsToCreate, workloadsToDelete)
}

func (g *GitWriter) update(subDir, workPlacementName string, workloadsToCreate []v1alpha1.Workload, workloadsToDelete []string) error {
	if len(workloadsToCreate) == 0 && len(workloadsToDelete) == 0 && subDir == "" {
		return nil
	}

	dirInGitRepo := filepath.Join(g.path, subDir)
	logger := g.Log.WithValues(
		"dir", dirInGitRepo,
		"branch", g.gitServer.Branch,
	)

	localTmpDir, repo, worktree, err := g.setupLocalDirectoryWithRepo(logger)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(localTmpDir))

	err = g.deleteExistingFiles(subDir != "", dirInGitRepo, workloadsToDelete, worktree, logger)
	if err != nil {
		return err
	}

	for _, file := range workloadsToCreate {
		//worker-cluster/resources/<rr-namespace>/<promise-name>/<rr-name>/foo/bar/baz.yaml
		worktreeFilePath := filepath.Join(dirInGitRepo, file.Filepath)
		log := logger.WithValues(
			"filepath", worktreeFilePath,
		)

		///tmp/git-dir/worker-cluster/resources/<rr-namespace>/<promise-name>/<rr-name>/foo/bar/baz.yaml
		absoluteFilePath := filepath.Join(localTmpDir, worktreeFilePath)

		//We need to protect against paths containing `..`
		//filepath.Join expands any '../' in the path to the actual, e.g. /tmp/foo/../ resolves to /tmp/
		//To ensure they can't write to files on disk outside the tmp git repository we check the absolute path
		//returned by `filepath.Join` is still contained with the git repository:
		// Note: This means `../` can still be used, but only if the end result is still contained within the git repository
		if !strings.HasPrefix(absoluteFilePath, localTmpDir) {
			log.Error(nil, "path of file to write is not located within the git repostiory", "absolutePath", absoluteFilePath, "tmpDir", localTmpDir)
			return nil //We don't want to retry as this isn't a recoverable error. Log error and return nil.
		}

		if err := os.MkdirAll(filepath.Dir(absoluteFilePath), 0700); err != nil {
			log.Error(err, "could not generate local directories")
			return err
		}

		if err := os.WriteFile(absoluteFilePath, []byte(file.Content), 0644); err != nil {
			log.Error(err, "could not write to file")
			return err
		}

		if _, err := worktree.Add(worktreeFilePath); err != nil {
			log.Error(err, "could not add file to worktree")
			return err
		}
	}

	action := "Delete"
	if len(workloadsToCreate) > 0 {
		action = "Update"
	}
	return g.commitAndPush(repo, worktree, action, workPlacementName, logger)
}

// deleteExistingFiles removes all files in dir when removeDirectory is set to true
// else it removes files listed in workloadsToDelete
func (g *GitWriter) deleteExistingFiles(removeDirectory bool, dir string, workloadsToDelete []string, worktree *git.Worktree, logger logr.Logger) error {
	if removeDirectory {
		if _, err := worktree.Filesystem.Lstat(dir); err == nil {
			logger.Info("deleting existing content")
			if _, err := worktree.Remove(dir); err != nil {
				logger.Error(err, "could not add directory deletion to worktree", "dir", dir)
				return err
			}
		}
	} else {
		for _, file := range workloadsToDelete {
			worktreeFilePath := filepath.Join(dir, file)
			log := logger.WithValues(
				"filepath", worktreeFilePath,
			)
			if _, err := worktree.Filesystem.Lstat(worktreeFilePath); err != nil {
				log.Info("file requested to be deleted from worktree but does not exist")
				continue
			}
			if _, err := worktree.Remove(worktreeFilePath); err != nil {
				logger.Error(err, "could not remove file from worktree")
				return err
			}
			logger.Info("successfully deleted file from worktree")
		}
	}
	return nil
}

func (g *GitWriter) ReadFile(filePath string) ([]byte, error) {
	logger := g.Log.WithValues(
		"path", filePath,
		"branch", g.gitServer.Branch,
	)

	localTmpDir, _, worktree, err := g.setupLocalDirectoryWithRepo(logger)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(filepath.Dir(localTmpDir))

	if _, err := worktree.Filesystem.Lstat(filePath); err != nil {
		return nil, FileNotFound
	}

	return os.ReadFile(filepath.Join(localTmpDir, filePath))
}

func (g *GitWriter) setupLocalDirectoryWithRepo(logger logr.Logger) (string, *git.Repository, *git.Worktree, error) {
	localTmpDir, err := createLocalDirectory(logger)
	if err != nil {
		logger.Error(err, "could not create temporary repository directory")
		return "", nil, nil, err
	}

	repo, err := g.cloneRepo(localTmpDir, logger)
	if err != nil {
		logger.Error(err, "could not clone repository")
		return "", nil, nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		logger.Error(err, "could not access repo worktree")
		return "", nil, nil, err
	}
	return localTmpDir, repo, worktree, nil
}

func (g *GitWriter) push(repo *git.Repository, logger logr.Logger) error {
	err := repo.Push(&git.PushOptions{
		RemoteName:      "origin",
		Auth:            g.gitServer.Auth,
		InsecureSkipTLS: true,
	})
	if err != nil {
		logger.Error(err, "could not push to remote")
		return err
	}
	return nil
}

func (g *GitWriter) cloneRepo(localRepoFilePath string, logger logr.Logger) (*git.Repository, error) {
	logger.Info("cloning repo")
	return git.PlainClone(localRepoFilePath, false, &git.CloneOptions{
		Auth:            g.gitServer.Auth,
		URL:             g.gitServer.URL,
		ReferenceName:   plumbing.NewBranchReferenceName(g.gitServer.Branch),
		SingleBranch:    true,
		Depth:           1,
		NoCheckout:      false,
		InsecureSkipTLS: true,
	})
}

func (g *GitWriter) commitAndPush(repo *git.Repository, worktree *git.Worktree, action, workPlacementName string, logger logr.Logger) error {
	status, err := worktree.Status()
	if err != nil {
		logger.Error(err, "could not get worktree status")
		return err
	}

	if status.IsClean() {
		logger.Info("no changes to be committed")
		return nil
	}

	_, err = worktree.Commit(fmt.Sprintf("%s from: %s", action, workPlacementName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  g.author.Name,
			Email: g.author.Email,
			When:  time.Now(),
		},
	})

	if err != nil {
		logger.Error(err, "could not commit file to worktree")
		return err
	}

	logger.Info("pushing changes")
	if err := g.push(repo, logger); err != nil {
		logger.Error(err, "could not push changes")
		return err
	}
	return nil
}

func createLocalDirectory(logger logr.Logger) (string, error) {
	logger.Info("creating local directory")
	dir, err := os.MkdirTemp("", "kratix-repo")
	if err != nil {
		return "", err
	}

	return dir, nil
}
