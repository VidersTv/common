package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CountDocument struct {
	ID     primitive.ObjectID `bson:"_id"`    // ObjectID		primary-key
	Key    interface{}        `bson:"key"`    // string			index(key)
	Group  interface{}        `bson:"group"`  // ObjectID		index(group)
	Type   CountDocumentType  `bson:"type"`   // string			index(type)
	Expiry time.Time          `bson:"expiry"` // Time			index(expiry) *ttl
}

type CountDocumentType string

const (
	CountDocumentTypeChatter CountDocumentType = "CHATTER"
	CountDocumentTypeViewer  CountDocumentType = "VIEWER"
)
