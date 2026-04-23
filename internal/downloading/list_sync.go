package downloading

import (
	"context"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/database"
)

type ListSyncManager struct {
	db *sqlx.DB
	mu sync.Mutex
}

func NewListSyncManager(db *sqlx.DB) *ListSyncManager {
	return &ListSyncManager{
		db: db,
	}
}

func (lsm *ListSyncManager) SyncListMembers(ctx context.Context, lstEntityId int, lstName string, currentMemberIDs []uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	lsm.mu.Lock()
	defer lsm.mu.Unlock()

	tx, err := lsm.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	links, err := database.GetUserLinksByLstEntityId(tx, lstEntityId)
	if err != nil {
		return err
	}

	memberSet := make(map[uint64]bool)
	for _, id := range currentMemberIDs {
		memberSet[id] = true
	}

	// 收集需要删除的链接ID
	var idsToRemove []int32
	for _, link := range links {
		if !memberSet[link.UserId] {
			// 先删除符号链接（文件系统操作不能在事务中批量处理）
			if linkpath, err := link.PathTx(tx); err == nil {
				if err := os.Remove(linkpath); err != nil && !os.IsNotExist(err) {
					log.Warnln("failed to remove symlink:", linkpath, err)
				}
			}
			idsToRemove = append(idsToRemove, link.Id)
		}
	}

	// 批量删除数据库记录
	if len(idsToRemove) > 0 {
		if err := database.DeleteUserLinksBatch(tx, idsToRemove); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if len(idsToRemove) > 0 {
		log.Infoln("Removed", len(idsToRemove), "users from list", lstName, "(no longer members)")
	}

	return nil
}
