package domains

import "github.com/globalsign/mgo"

func (o *ErrorHandler) EnsuredMongoIndexes(db *mgo.Database) {
	o.EnsureMessageIndexes(db)
	if o.Err != nil {
		return
	}

	o.EnsureChatRoomIndexes(db)
	if o.Err != nil {
		return
	}
}
