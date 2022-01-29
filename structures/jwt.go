package structures

import (
	"github.com/golang-jwt/jwt"
	"github.com/viderstv/common/errors"
	"github.com/viderstv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JwtTranscodePayload struct {
	StreamID        primitive.ObjectID `json:"stream_id"`
	UserID          primitive.ObjectID `json:"user_id"`
	Revision        int32              `json:"revision"`
	TranscodeStream bool               `json:"transcode_stream"`
	IngestPodIP     string             `json:"ingest_pod_ip"`
	jwt.StandardClaims
}

type JwtMuxerPayload struct {
	StreamID        primitive.ObjectID     `json:"stream_id"`
	UserID          primitive.ObjectID     `json:"user_id"`
	Variant         JwtMuxerPayloadVariant `json:"variant"`
	IngestPodIP     string                 `json:"ingest_pod_ip"`
	TranscoderPodIP string                 `json:"transcoder_pod_ip"`
	jwt.StandardClaims
}

type JwtMuxerPayloadVariant struct {
	Name    string `json:"name"`
	Codecs  string `json:"codecs"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	FPS     int    `json:"fps"`
	Bitrate int    `json:"bitrate"`
}

type JwtInternalRead struct {
	StreamID primitive.ObjectID `json:"stream_id"`
	jwt.StandardClaims
}

func EncodeJwt(claims jwt.Claims, key string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(utils.S2B(key))
}

func DecodeJwt(claims jwt.Claims, key string, token string) error {
	tkn, err := jwt.ParseWithClaims(token, claims, func(tkn *jwt.Token) (interface{}, error) {
		if _, ok := tkn.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.ErrJwtTokenInvalid
		}

		return utils.S2B(key), nil
	})
	if err != nil {
		return err
	}

	if !tkn.Valid {
		return errors.ErrJwtTokenInvalid
	}

	return nil
}
