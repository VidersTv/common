package structures

import "go.mongodb.org/mongo-driver/bson/primitive"

// Message structure is a MongoDB object in the schema "messages"
type Message struct {
	ID        primitive.ObjectID `bson:"id"`         // ObjectID		primary-key
	UserID    primitive.ObjectID `bson:"user_id"`    // ObjectID		index(user_id),		index(user_id, channel_id)
	ChannelID primitive.ObjectID `bson:"channel_id"` // ObjectID		index(channel_id),	index(user_id, channel_id)
	Content   string             `bson:"content"`    // string			index-text(content)
	Emotes    []MessageEmote     `bson:"emotes"`     // ChatEmote
}

// MessageEmote structure is a MongoDB object in the object `Message` in the schema "messages"
type MessageEmote struct {
	ID  primitive.ObjectID `bson:"id"`  // ObjectID
	Tag string             `bson:"tag"` // string
}
