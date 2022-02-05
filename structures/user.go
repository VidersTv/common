package structures

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User structure is a MongoDB object in the schema "users"
type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`  // ObjectID		primary-key
	Login         string             `bson:"login"`          // string			index(login)
	DisplayName   string             `bson:"display_name"`   // string
	Color         Color              `bson:"color"`          // int32
	Role          GlobalRole         `bson:"role"`           // int32			index(role)
	Channel       Channel            `bson:"channel"`        // Channel
	TwitchAccount TwitchAccount      `bson:"twitch_account"` // TwitchAccount
	Memberships   []Member           `bson:"memberships"`    // []Member
}

func (u User) MemberRole(channelID primitive.ObjectID) ChannelRole {
	if channelID == u.ID {
		return ChannelRoleAdmin
	}

	for _, v := range u.Memberships {
		if v.ChannelID == channelID {
			return v.Role
		}
	}

	return ChannelRoleUser
}
