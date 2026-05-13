package handlers

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwksMu       sync.Mutex
	jwksKeyCache map[string]interface{}
	jwksCacheAt  time.Time
	jwksCacheTTL = 30 * time.Minute
)

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

func parseRSAPublicKey(nB64, eB64 string) (interface{}, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}

	// Convert exponent bytes to int.
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid RSA exponent")
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}
	return pub, nil
}

func fetchJWKSKeys(url string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch JWKS")
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	keys := make(map[string]interface{})
	for _, k := range doc.Keys {
		if k.Kid == "" || k.Kty != "RSA" || k.N == "" || k.E == "" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	if len(keys) == 0 {
		return nil, errors.New("no usable RSA keys in JWKS")
	}
	return keys, nil
}

func getJWKSKey(url, kid string) (interface{}, error) {
	jwksMu.Lock()
	defer jwksMu.Unlock()

	if jwksKeyCache != nil && time.Since(jwksCacheAt) < jwksCacheTTL {
		if key, ok := jwksKeyCache[kid]; ok {
			return key, nil
		}
	}

	keys, err := fetchJWKSKeys(url)
	if err != nil {
		return nil, err
	}
	jwksKeyCache = keys
	jwksCacheAt = time.Now()

	key, ok := keys[kid]
	if !ok {
		return nil, errors.New("kid not found in JWKS")
	}
	return key, nil
}

// AuthMiddleware validates a Supabase/Gotrue JWT and ensures a user exists in the local users table.
// It supports either HS256 (via SUPABASE_JWT_SECRET) or RS256 via JWKS (SUPABASE_JWKS_URL or derived from SUPABASE_URL).
func AuthMiddleware(userService *services.UserService) gin.HandlerFunc {
	// Precompute configuration from environment
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	jwksURL := os.Getenv("SUPABASE_JWKS_URL")
	if jwksURL == "" {
		if sup := os.Getenv("SUPABASE_URL"); sup != "" {
			jwksURL = strings.TrimRight(sup, "/") + "/auth/v1/.well-known/jwks.json"
		}
	}

	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		tokenStr := strings.TrimSpace(auth[len("Bearer "):])

		var token *jwt.Token
		var err error

		if secret != "" {
			token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				// Expect HS256 when using secret
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, jwt.ErrTokenUnverifiable
				}
				return []byte(secret), nil
			})
		} else if jwksURL != "" {
			token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
					return nil, jwt.ErrTokenUnverifiable
				}
				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, errors.New("missing kid")
				}
				return getJWKSKey(jwksURL, kid)
			})
			if err != nil {
				log.Printf("JWKS token parse failed: %v", err)
			}
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "no JWT verifier configured"})
			return
		}

		if err != nil || token == nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Extract the subject (user id). Supabase uses `sub`.
		sub, _ := claims["sub"].(string)
		if sub == "" {
			// Some tokens may use `user_id` claim.
			sub, _ = claims["user_id"].(string)
		}
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing subject"})
			return
		}

		// Extract email and display name from token claims if available
		var email string
		var displayName string
		if e, ok := claims["email"].(string); ok {
			email = e
		}
		if n, ok := claims["name"].(string); ok {
			displayName = n
		} else if email != "" {
			displayName = email
		}

		// Ensure user exists in database (idempotent). Pass email so email is saved for email/password signups.
		_, _ = userService.CreateUser(context.Background(), sub, displayName, email)

		// Attach user id to context for downstream handlers
		c.Set("user_id", sub)

		c.Next()
	}
}
