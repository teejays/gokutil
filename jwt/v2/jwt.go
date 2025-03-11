package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/scalars"
)

var ErrMalformedToken = fmt.Errorf("malformed token")
var ErrBadSignature = fmt.Errorf("bad signature")
var ErrTokenTimestamp = fmt.Errorf("token is expired or not valid yet")

// New

type Claim[T any] struct {
	UserID       scalars.ID `json:"sub"`
	KeyReference string     `json:"keyReference"` // Identifier for the key used to sign the token
	OtherData    T          `json:"otherData"`
	jwt.RegisteredClaims
}

type GenerateTokenRequest[T any] struct {
	UserID            scalars.ID
	OtherData         T `json:"otherData"`
	Duration          time.Duration
	Now               scalars.Timestamp
	KeyReference      string
	GetPrivateKeyFunc func(ctx context.Context, keyReference string) (*rsa.PrivateKey, error)
}

func (r GenerateTokenRequest[T]) Validate() error {
	if err := r.UserID.Validate(); err != nil {
		return errors.New("userID is invalid")
	}
	if r.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}
	if r.Duration < 5*time.Second {
		return errors.New("duration must be more than 5 seconds")
	}
	if r.GetPrivateKeyFunc == nil {
		return errors.New("GetPrivateKeyFunc must not be nil")
	}
	return nil
}

func GenerateSignedToken[T any](ctx context.Context, req GenerateTokenRequest[T]) ([]byte, error) {
	now := scalars.NewTimestampNow()

	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, errutil.Wrap(err, "validating request")
	}

	// Use from the request if provided
	if !req.Now.IsEmpty() {
		now = req.Now
	}

	// What key to use?

	// Get the private key
	key, err := req.GetPrivateKeyFunc(ctx, req.KeyReference)
	if err != nil {
		return nil, errutil.Wrap(err, "getting private key")
	}

	// Generate the token
	log.Info(ctx, "Generating the token.")
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, Claim[T]{
		UserID:       req.UserID,
		KeyReference: req.KeyReference,
		OtherData:    req.OtherData,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   req.UserID.String(),
			Issuer:    "ongoku-based-api-server",
			IssuedAt:  jwt.NewNumericDate(now.ToGolangTime()),
			NotBefore: jwt.NewNumericDate(now.ToGolangTime()),
			ExpiresAt: jwt.NewNumericDate(now.ToGolangTime().Add(req.Duration)),
		},
	})

	log.Info(ctx, "Signing the token.")
	signedToken, err := token.SignedString(key)
	if err != nil {
		return nil, errutil.Wrap(err, "signing the token")
	}

	log.Trace(ctx, "Generated token", "token", signedToken)

	return []byte(signedToken), nil
}

type ValidateTokenRequest struct {
	TokenString      string
	GetPublicKeyFunc func(ctx context.Context, keyReference string) (*rsa.PublicKey, error)
}

func ValidateSignedToken[T any](ctx context.Context, req ValidateTokenRequest) (*Claim[T], error) {
	var err error
	log.Debug(ctx, "Validating the token.")

	if req.TokenString == "" {
		return nil, errors.New("token is empty")
	}
	if req.GetPublicKeyFunc == nil {
		return nil, errors.New("GetPublicKeyFunc must not be nil")
	}

	// Before we validate the token, we need to get the public key reference from the token itself
	// and then get the public key from the file
	log.Debug(ctx, "Parsing the token unverified to extract the key reference.")
	var unverifiedClaims Claim[T]
	jwtParser := jwt.Parser{}
	_, _, err = jwtParser.ParseUnverified(req.TokenString, &unverifiedClaims)
	if err != nil {
		return nil, errutil.Wrap(err, "parsing unverified token")
	}
	if unverifiedClaims.KeyReference == "" {
		return nil, errors.New("key reference not found in the token")
	}
	keyReference := unverifiedClaims.KeyReference

	log.Debug(ctx, "Key reference from the token", "keyReference", keyReference)

	// Parse the token
	log.Trace(ctx, "Parsing the token.")
	var verifiedClaims Claim[T]
	token, err := jwt.ParseWithClaims(req.TokenString, &verifiedClaims, func(t *jwt.Token) (interface{}, error) {
		// Get the public key
		log.Debug(ctx, "Getting the public key.")
		pubKey, err := req.GetPublicKeyFunc(ctx, keyReference)
		if err != nil {
			return nil, errutil.Wrap(err, "getting public key")
		}

		return pubKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrTokenTimestamp
	}

	// Check the claims
	return &verifiedClaims, nil
}
