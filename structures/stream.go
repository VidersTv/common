package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream structure is a MongoDB object in the schema "streams"
type Stream struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` // ObjectID		primary-key
	UserID    primitive.ObjectID `bson:"user_id"`       // ObjectID		index(user_id)
	Title     string             `bson:"title"`         // stirng
	StartedAt time.Time          `bson:"started_at"`    // time
	EndedAt   time.Time          `bson:"ended_at"`      // time
	Revision  int32              `bson:"revision"`      // int32
}
