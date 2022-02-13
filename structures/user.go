package structures

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User structure is a MongoDB object in the schema "users"
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`               // ObjectID		primary-key
	Login          string             `bson:"login" json:"login,omitempty"`                     // string			index(login)
	ProfilePicture primitive.ObjectID `bson:"profile_picture" json:"profile_picture,omitempty"` // ObjectID
	DisplayName    string             `bson:"display_name" json:"display_name,omitempty"`       // string
	Color          Color              `bson:"color" json:"color,omitempty"`                     // int32
	Role           GlobalRole         `bson:"role" json:"role,omitempty"`                       // int32			index(role)
	Channel        Channel            `bson:"channel" json:"channel,omitempty"`                 // Channel
	TwitchAccount  TwitchAccount      `bson:"twitch_account" json:"twitch_account,omitempty"`   // TwitchAccount
	Memberships    []Member           `bson:"memberships" json:"memberships,omitempty"`         // []Member
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
