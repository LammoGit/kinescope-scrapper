package services

import (
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Errors
var (
	ErrInvalidToken     = utility.NewValidationError("Received token is invalid")
	ErrFailedToReadSalt = utility.NewInternalError("Crypto service failed to write random bytes to a salt buffer")
	ErrEmptyIssuer      = utility.NewInternalError("Empty issuer is given to the crypto service")
	ErrEmptySecret      = utility.NewInternalError("Empty secret is given to the crypto service")
)

// UserClaims claims of user's JWT token
type UserClaims struct {
	UserID   int             `json:"user_id"`   // User's id
	UserRole domain.UserRole `json:"user_role"` // User's role
	Type     string          `json:"typ"`       // Type of token: "access" or "refresh"
	jwt.RegisteredClaims
}

// Argon2Config configuration of argon2 hashing algorithm
type Argon2Config struct {
	Memory     uint32 // Memory cost of computing a hash in KiB
	Iterations uint32 // Number of passes over the Memory
	Threads    uint8  // Number of threads used
	KeyLen     uint32 // Length of a generated hash
	SaltSize   int    // Length of random salt used for hashing
}

// CryptoOption functions ran on CryptoService object at construction
type CryptoOption func(s *CryptoService)

// WithArgonMemory CryptoService option that sets memory used by Argon2 in KiB
func WithArgonMemory(mem uint32) CryptoOption {
	return func(s *CryptoService) {
		if mem != 0 {
			s.argonConfig.Memory = mem
		}
	}
}

// WithArgonIterations CryptoService option that sets number of passes in Argon2
func WithArgonIterations(iter uint32) CryptoOption {
	return func(s *CryptoService) {
		if iter != 0 {
			s.argonConfig.Iterations = iter
		}
	}
}

// WithArgonThreads CryptoService option that sets number of threads used for Argon2 calculation
func WithArgonThreads(threads uint8) CryptoOption {
	return func(s *CryptoService) {
		if threads != 0 {
			s.argonConfig.Threads = threads
		}
	}
}

// WithKeyLen CryptoService option that sets length of a password hash generated
func WithKeyLen(keyLen uint32) CryptoOption {
	return func(s *CryptoService) {
		if keyLen != 0 {
			s.argonConfig.KeyLen = keyLen
		}
	}
}

// WithSaltSize CryptoService option that sets length of a random salt generated for hashing
func WithSaltSize(saltSize int) CryptoOption {
	return func(s *CryptoService) {
		if saltSize > 0 {
			s.argonConfig.SaltSize = saltSize
		}
	}
}

// WithArgonConfig CryptoService option that sets full Argon2 hashing algorithm config without validation
func WithArgonConfig(config Argon2Config) CryptoOption {
	return func(s *CryptoService) {
		s.argonConfig = config
	}
}

// WithAccessTTL CryptoService option that sets TTL of an access token
func WithAccessTTL(ttl time.Duration) CryptoOption {
	return func(s *CryptoService) {
		s.accessTTL = ttl
	}
}

// WithRefreshTTL CryptoService option that sets TTL of a refresh token
func WithRefreshTTL(ttl time.Duration) CryptoOption {
	return func(s *CryptoService) {
		s.refreshTTL = ttl
	}
}

// CryptoService object used for managing passwords and JWT
type CryptoService struct {
	secret      []byte        // Secret used by password hashing algorithm
	issuer      string        // Issuer set in generated JWT tokens
	argonConfig Argon2Config  // Config of Argon2 password hashing algorithm
	accessTTL   time.Duration // TTL of generated access tokens
	refreshTTL  time.Duration // TTL of generated refresh tokens
}

// NewCryptoService returns new CryptoService with set secret and issuer, and with applies options to it
func NewCryptoService(secret []byte, issuer string, opts ...CryptoOption) (*CryptoService, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}
	if issuer == "" {
		return nil, ErrEmptyIssuer
	}

	service := &CryptoService{
		secret: secret,
		issuer: issuer,
		argonConfig: Argon2Config{
			Memory:     64 * 1024,
			Iterations: 3,
			Threads:    2,
			KeyLen:     32,
			SaltSize:   16,
		},
		accessTTL:  30 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service, nil
}

// HashPassword hashes user password and returns hash with random salt used during hash generation
func (s *CryptoService) HashPassword(password domain.Password) (salt, hash []byte, err error) {
	salt = make([]byte, s.argonConfig.SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, ErrFailedToReadSalt
	}
	hash = argon2.IDKey(
		[]byte(password),
		salt,
		s.argonConfig.Iterations,
		s.argonConfig.Memory,
		s.argonConfig.Threads,
		s.argonConfig.KeyLen,
	)
	return salt, hash, nil
}

// VerifyPassword check whether given password matches salt-hash pair
func (s *CryptoService) VerifyPassword(password string, salt, hash []byte) bool {
	if len(salt) == 0 || len(hash) == 0 {
		return false
	}
	computed := argon2.IDKey(
		[]byte(password),
		salt,
		s.argonConfig.Iterations,
		s.argonConfig.Memory,
		s.argonConfig.Threads,
		uint32(len(hash)),
	)

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(computed, hash) == 1
}

// sign private function used to sign one token of given type, and with given duration
func (s *CryptoService) sign(user domain.User, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserID:   user.ID,
		UserRole: user.Role,
		Type:     typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// SignTokens returns signed acess and refresh JWT tokens in form of strings
func (s *CryptoService) SignTokens(user domain.User) (access, refresh string, err error) {
	access, err = s.sign(user, "access", s.accessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = s.sign(user, "refresh", s.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// ParseToken parses JWT token string into UserClaims
func (s *CryptoService) ParseToken(tokenString string) (*UserClaims, error) {
	var claims UserClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid || claims.Issuer != s.issuer {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}
