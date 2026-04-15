package database

import (
	"github.com/jmoiron/sqlx"
)

func SyncUser(db *sqlx.DB, userId uint64, name string, screenName string, isProtected bool, friendsCount int, accessible bool) error {
	renamed := false
	isNew := false
	usrdb, err := GetUserById(db, userId)
	if err != nil {
		return err
	}

	if usrdb == nil {
		isNew = true
		usrdb = &User{}
		usrdb.Id = userId
	} else {
		renamed = usrdb.Name != name || usrdb.ScreenName != screenName
	}

	usrdb.FriendsCount = friendsCount
	usrdb.IsProtected = isProtected
	usrdb.Name = name
	usrdb.ScreenName = screenName
	usrdb.IsAccessible = accessible

	if isNew {
		err = CreateUser(db, usrdb)
	} else {
		err = UpdateUser(db, usrdb)
	}
	if err != nil {
		return err
	}
	if renamed || isNew {
		return RecordUserPreviousName(db, userId, name, screenName)
	}
	return nil
}
