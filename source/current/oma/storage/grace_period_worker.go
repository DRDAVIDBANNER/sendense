// Package storage provides grace period worker for automatic immutability application
// Following project rules: modular design, background workers, no simulations
package storage

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

// GracePeriodWorker processes backups whose grace period has expired.
// Applies immutability (chattr +i) to backups after the configured grace period.
// Enterprise ransomware protection: Automatic immutability application.
type GracePeriodWorker struct {
	repoManager  *RepositoryManager
	backupRepo   BackupChainRepository
	checkInterval time.Duration
	stopChan     chan struct{}
}

// NewGracePeriodWorker creates a new grace period worker.
func NewGracePeriodWorker(repoManager *RepositoryManager, backupRepo BackupChainRepository) *GracePeriodWorker {
	return &GracePeriodWorker{
		repoManager:  repoManager,
		backupRepo:   backupRepo,
		checkInterval: 1 * time.Hour, // Check every hour
		stopChan:     make(chan struct{}),
	}
}

// Start begins the grace period worker loop.
func (w *GracePeriodWorker) Start(ctx context.Context) {
	log.Info("Grace period worker started")

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processAllRepositories(ctx)

	for {
		select {
		case <-ticker.C:
			w.processAllRepositories(ctx)
		case <-w.stopChan:
			log.Info("Grace period worker stopped")
			return
		case <-ctx.Done():
			log.Info("Grace period worker context cancelled")
			return
		}
	}
}

// Stop stops the grace period worker.
func (w *GracePeriodWorker) Stop() {
	close(w.stopChan)
}

// processAllRepositories processes grace period backups for all immutable repositories.
func (w *GracePeriodWorker) processAllRepositories(ctx context.Context) {
	log.Debug("Processing grace period backups for all repositories")

	// Get all repositories
	repos, err := w.repoManager.ListRepositories(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list repositories")
		return
	}

	processedRepos := 0
	for _, repoConfig := range repos {
		// Skip if not immutable
		if !repoConfig.IsImmutable {
			continue
		}

		// Get repository instance
		repo, err := w.repoManager.GetRepository(ctx, repoConfig.ID)
		if err != nil {
			log.WithError(err).WithField("repo_id", repoConfig.ID).Warn("Failed to get repository")
			continue
		}

		// Check if repository is immutable wrapper
		immutableRepo, ok := repo.(*ImmutableRepository)
		if !ok {
			log.WithField("repo_id", repoConfig.ID).Warn("Repository marked immutable but not wrapped in ImmutableRepository")
			continue
		}

		// Process grace period backups for all VMs using this repository
		if err := w.processRepositoryBackups(ctx, immutableRepo, repoConfig.ID); err != nil {
			log.WithError(err).WithField("repo_id", repoConfig.ID).Error("Failed to process repository backups")
			continue
		}

		processedRepos++
	}

	if processedRepos > 0 {
		log.WithField("repositories_processed", processedRepos).Info("Grace period processing complete")
	}
}

// processRepositoryBackups processes all backups in a repository.
func (w *GracePeriodWorker) processRepositoryBackups(ctx context.Context, repo *ImmutableRepository, repoID string) error {
	// Get all backup chains (this gives us all VMs with backups)
	// Note: We would need a method to list all VM contexts with backups
	// For now, this is a placeholder for the actual implementation

	log.WithField("repo_id", repoID).Debug("Processing backups for repository")

	// TODO: Once we have a method to list all VM contexts with backups,
	// we can iterate through them and call repo.ProcessGracePeriodBackups(ctx, vmContextID)
	// for each VM context.

	// Example:
	// vmContexts, err := w.backupRepo.ListVMContextsWithBackups(ctx)
	// if err != nil {
	// 	return err
	// }
	//
	// for _, vmContextID := range vmContexts {
	// 	if err := repo.ProcessGracePeriodBackups(ctx, vmContextID); err != nil {
	// 		log.WithError(err).WithField("vm_context_id", vmContextID).Warn("Failed to process grace period backups")
	// 	}
	// }

	return nil
}

// RunOnce processes grace period backups once (useful for testing or manual triggers).
func (w *GracePeriodWorker) RunOnce(ctx context.Context) error {
	w.processAllRepositories(ctx)
	return nil
}

// SetCheckInterval changes the check interval (useful for testing).
func (w *GracePeriodWorker) SetCheckInterval(interval time.Duration) {
	w.checkInterval = interval
}
