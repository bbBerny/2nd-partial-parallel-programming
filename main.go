package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Album represents data about a record album.
type Album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

var (
	// Data stores
	albums   []Album
	sessions map[string]string // token -> username
	
	// Mutexes for thread safety
	albumsMutex   sync.RWMutex
	sessionsMutex sync.RWMutex
)

// Initialize data stores
func init() {
	albums = []Album{
		{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
		{ID: "2", Title: "Time Out", Artist: "Dave Brubeck", Price: 37.99},
		{ID: "3", Title: "Flying Beagle", Artist: "Himiko Kikuchi", Price: 69.99},
	}
	sessions = make(map[string]string)
}

// Generate a secure random token
func generateToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// AuthMiddleware checks if the request has a valid Bearer token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid Authorization header format"})
			return
		}

		token := parts[1]

		sessionsMutex.RLock()
		username, exists := sessions[token]
		sessionsMutex.RUnlock()

		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			return
		}

		// Store username in context for later use
		c.Set("username", username)
		c.Next()
	}
}

// login handles user authentication
func login(c *gin.Context) {
	// For this project, any non-empty username/password is accepted via Basic Auth
	username, password, hasAuth := c.Request.BasicAuth()
	if !hasAuth || username == "" || password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Valid Basic Authentication required"})
		return
	}

	token := generateToken()

	sessionsMutex.Lock()
	sessions[token] = username
	sessionsMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Hi " + username + ", welcome to the Store System",
		"token":   token,
	})
}

// logout revokes the user's token
func logout(c *gin.Context) {
	// The AuthMiddleware already extracted the token and verified it
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	token := parts[1]

	username := c.GetString("username")

	sessionsMutex.Lock()
	delete(sessions, token)
	sessionsMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Bye " + username + ", your token has been revoked",
	})
}

// getAlbums responds with the list of all albums as JSON.
func getAlbums(c *gin.Context) {
	albumsMutex.RLock()
	defer albumsMutex.RUnlock()
	c.JSON(http.StatusOK, albums)
}

// getAlbumByID locates the album whose ID value matches the id parameter sent by the client.
func getAlbumByID(c *gin.Context) {
	id := c.Param("id")

	albumsMutex.RLock()
	defer albumsMutex.RUnlock()

	// Using a slice to format the response as specified in the assignment array
	for _, a := range albums {
		if a.ID == id {
			c.JSON(http.StatusOK, []Album{a})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
}

// postAlbum adds an album from JSON received in the request body.
func postAlbum(c *gin.Context) {
	var newAlbum Album

	// Call BindJSON to bind the received JSON to newAlbum.
	if err := c.BindJSON(&newAlbum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	albumsMutex.Lock()
	defer albumsMutex.Unlock()

	// Check for duplicate product IDs
	for _, a := range albums {
		if a.ID == newAlbum.ID {
			c.JSON(http.StatusConflict, gin.H{"message": "album with this ID already exists"})
			return
		}
	}

	// Add the new album to the slice.
	albums = append(albums, newAlbum)
	c.JSON(http.StatusCreated, newAlbum)
}

// status gets the overall system status and logged user details
func status(c *gin.Context) {
	username := c.GetString("username")
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	c.JSON(http.StatusOK, gin.H{
		"message": "Hi " + username + ", the DPIP System is Up and Running",
		"time":    currentTime,
	})
}

func main() {
	router := gin.Default()

	// Public routes
	router.GET("/login", login)

	// Protected routes
	protected := router.Group("/")
	protected.Use(AuthMiddleware())
	{
		protected.GET("/logout", logout)
		protected.GET("/albums", getAlbums)
		protected.GET("/albums/:id", getAlbumByID)
		protected.POST("/post-album", postAlbum)
		protected.GET("/status", status)
	}

	// Run the server
	router.Run("localhost:8080")
}
