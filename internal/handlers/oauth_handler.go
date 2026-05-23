package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ShamalLakshan/SwaRupa/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubOAuthConfig holds the OAuth2 configuration for GitHub.
// This is initialized once at startup in main.go.
var GitHubOAuthConfig *oauth2.Config

// InitGitHubOAuth initializes the GitHub OAuth2 configuration.
// Call this in main.go during application startup.
// Requires environment variables: GITHUB_OAUTH_CLIENT_ID, GITHUB_OAUTH_CLIENT_SECRET
// CALLBACK_URL should be set to your deployed domain (e.g., https://api.example.com/auth/github/callback)
func InitGitHubOAuth(redirectURL string) error {
	clientID := os.Getenv("GITHUB_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_OAUTH_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("GITHUB_OAUTH_CLIENT_ID and GITHUB_OAUTH_CLIENT_SECRET must be set")
	}

	GitHubOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email", "read:user"},
		Endpoint:     github.Endpoint,
	}

	return nil
}

// GitHubOAuthLogin handles GET /auth/github
// Redirects user to GitHub's OAuth consent screen.
// The state parameter prevents CSRF attacks by validating the callback.
func GitHubOAuthLogin(c *gin.Context) {
	if GitHubOAuthConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured"})
		return
	}

	// Generate a random state token (in production, store this in session/cache temporarily)
	// For now, use a simple timestamp-based state (not production-safe without session storage)
	state := fmt.Sprintf("%d", 12345) // TODO: use session middleware for proper CSRF protection

	// Redirect to GitHub's authorization endpoint
	authURL := GitHubOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GitHubUser represents the minimal user data from GitHub API
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

// GitHubOAuthCallback handles GET /auth/github/callback
// Exchanges the authorization code for an access token,
// fetches user data from GitHub, and creates/updates the local user record.
// Returns a JWT token for subsequent authenticated requests.
func GitHubOAuthCallback(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if GitHubOAuthConfig == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured"})
			return
		}

		// Extract authorization code from callback
		code := c.Query("code")
		state := c.Query("state")

		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
			return
		}

		// TODO: Validate state parameter against session to prevent CSRF
		// For now, basic validation passes
		if state == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing state parameter"})
			return
		}

		// Exchange authorization code for access token
		token, err := GitHubOAuthConfig.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to exchange authorization code"})
			return
		}

		// Fetch user info from GitHub API using the access token
		client := GitHubOAuthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch GitHub user data"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to authenticate with GitHub"})
			return
		}

		var ghUser GitHubUser
		if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse GitHub user data"})
			return
		}

		// Fetch user's email if not provided in the user endpoint
		if ghUser.Email == "" {
			emailResp, err := client.Get("https://api.github.com/user/emails")
			if err == nil {
				defer emailResp.Body.Close()
				var emails []struct {
					Email   string `json:"email"`
					Primary bool   `json:"primary"`
				}
				json.NewDecoder(emailResp.Body).Decode(&emails)
				for _, e := range emails {
					if e.Primary {
						ghUser.Email = e.Email
						break
					}
				}
			}
		}

		// Create or update user in database
		// This uses an idempotent UPSERT pattern: if github_id exists, update; else insert
		var contactEmail *string
		if ghUser.Email != "" {
			contactEmail = &ghUser.Email
		}
		user, err := userService.CreateOrUpdateUserFromGitHub(
			context.Background(),
			fmt.Sprintf("%d", ghUser.ID),
			ghUser.Login,
			ghUser.Name,
			ghUser.AvatarURL,
			contactEmail,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create or update user"})
			return
		}

		// TODO: Generate JWT token containing user_id and role
		// For now, return user ID (Phase 8 will add proper JWT generation)
		c.JSON(http.StatusOK, gin.H{
			"user_id":         user.ID,
			"username":        user.DisplayName,
			"github_username": ghUser.Login,
			// "token": jwtToken, // Add after implementing JWT middleware
		})
	}
}
