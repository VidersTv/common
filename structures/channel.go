package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Channel structure is a MongoDB object in the object `User` which is in the schema "users"
type Channel struct {
	Title            string    `bson:"title"`              // string
	Public           bool      `bson:"public"`             // boolean
	StreamKey        string    `bson:"stream_key"`         // string
	LastLive         time.Time `bson:"last_live"`          // time
	TwitchRoleMirror bool      `bson:"twitch_role_mirror"` // boolean
	Emotes           []Emote   `bson:"emotes"`             // Emote
}

// Member structure is a MongoDB object in the object `User` which is in the schema "users"
type Member struct {
	ChannelID primitive.ObjectID `bson:"channel_id"`  // ObjectID		index(channel_id, user_id), index(channel_id, role)
	Role      ChannelRole        `bson:"role"`        // int32			index(channel_id, role)
	AddedByID primitive.ObjectID `bson:"added_by_id"` // ObjectID 		index(channel_id, added_by_id)
}

// ChannelRole is a int32 value which denotes your permissions
type ChannelRole int32

const (
	// The default role
	ChannelRoleUser ChannelRole = iota
	// Can watch the channel if set to private
	ChannelRoleViewer
	// Extra permissions
	ChannelRoleVIP
	// Manage Emotes, title, category
	ChannelRoleEditor
	// Moderation permissions can manage chat, other users, and the channel
	ChannelRoleModerator
	// Admin permissions can manage channel, and other moderators
	ChannelRoleAdmin
)
