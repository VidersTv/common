package structures

import "go.mongodb.org/mongo-driver/bson/primitive"

// TwitchAccount structure is a MongoDB object in the object `User` which is in the schema "users"
type TwitchAccount struct {
	ID             string `bson:"id"`              // string		index-unique(id)
	Login          string `bson:"login"`           // string		index(login)
	DisplayName    string `bson:"display_name"`    // string
	ProfilePicture string `bson:"profile_picture"` // string
}

// TwitchRole structure is a MongoDB object in the schema "twitch_roles"
type TwitchRole struct {
	ID        primitive.ObjectID `bson:"_id"`
	ChannelID primitive.ObjectID `bson:"channel_id"`
	Type      TwitchRoleType     `bson:"type"`
}

// TwitchRoleType is an int32 which contains infomation about the user's twitch role
type TwitchRoleType int32

const (
	// If the user is a twitch Subscriber
	TwitchRoleTypeSub TwitchRoleType = iota
	// If the user is a twitch VIP
	TwitchRoleTypeVIP
	// If the user is a twitch mod
	TwitchRoleTypeMod
)
