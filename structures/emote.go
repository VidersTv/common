package structures

import "go.mongodb.org/mongo-driver/bson/primitive"

// Emote structure is a MongoDB object in the object `Channel` which is in the schema "users"
type Emote struct {
	ID         primitive.ObjectID `bson:"id" json:"id"`                   // ObjectID		index-unique(id)
	Tag        string             `bson:"tag" json:"tag"`                 // string
	UploaderID primitive.ObjectID `bson:"uploader_id" json:"uploader_id"` // ObjectID		index(uploader_id)
}
