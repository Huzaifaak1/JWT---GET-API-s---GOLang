package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	// "fmt"
	"log"

	"github.com/dgrijalva/jwt-go"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
)

var db *sql.DB
var jwtSecret = []byte("3cfa76ef14937c1c0ea519f8fc057a80fcd04a7420f8e8bcd0a7567c272e007b") // Replace with your secret key

func main() {
	// Define the connection string
	connString := "sqlserver://sa:root@localhost:1433?database=blog_be"

	// Open a connection to the database
	var err error
	db, err = sql.Open("mssql", connString)
	if err != nil {
		log.Fatalf("error opening connection: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("error pinging database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Set up Gin router
	router := gin.Default()

	// Apply middlware
	router.Use(AuthMiddleware())

	// Define routes
	router.GET("/api/v1/posts", GetPosts)
	router.GET("/api/v1/users", GetUsers)
	router.GET("/api/v1/user/posts/:id", getUserPostsAll)

	// Start the server
	router.Run(":8080")
}

// Middleware for JWT and restricting the routes
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		log.Println("Header: ", authHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		var length = len(tokenString)
		log.Println("Length: %v", length)
		// log.Println("token:**%v**", tokenString)
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
			// return []byte(jwtSecret), nil

		})

		if err != nil {
			log.Printf("Error parsing token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetPosts handles GET requests to fetch posts from the database
func GetPosts(c *gin.Context) {
	rows, err := db.Query("SELECT p.id, title, description, u.name FROM posts p LEFT JOIN users u ON u.id=p.user_id")
	if err != nil {
		// log.Println("error in query: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error executing query"})
		return
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id int
		var title string
		var description string
		var name string
		if err := rows.Scan(&id, &title, &description, &name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning row"})
			return
		}
		post := map[string]interface{}{
			"post_id":     id,
			"title":       title,
			"description": description,
			"name":        name,
		}
		posts = append(posts, post)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating over rows"})
		return
	}

	c.JSON(http.StatusOK, posts)
}
func GetUsers(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error executing query"})
	}
	defer rows.Close() // Correctly defer closing the rows, not the database connection

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		var email string

		if err := rows.Scan(&id, &name, &email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while scanning"})
			return
		}

		user := map[string]interface{}{
			"user_id": id,
			"name":    name,
			"email":   email,
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot iterate over the User."})
		return
	}
	c.JSON(http.StatusOK, users)
}

func getUserPostsAll(c *gin.Context) {
	type GetUserPostsResponse struct {
		TotalCount int                      `json:"total_count"`
		Posts      []map[string]interface{} `json:"posts"`
	}
	log.Println("Here")
	userId := c.Param("id")

	var totalCount int
	countQuery := "SELECT COUNT(*) FROM posts WHERE user_id = ?"
	err := db.QueryRow(countQuery, userId).Scan(&totalCount)
	if err != nil {
		log.Printf("Error getting post count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting post count"})
		return
	}

	rows, err := db.Query("SELECT u.id, u.name,p.title,p.description,p.id from posts p left join users u on u.id = p.user_id where p.user_id = ?", userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error in query"})
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var userID int
		var userName string
		var postTitle string
		var postDescription string
		var postID int

		if err := rows.Scan(&userID, &userName, &postTitle, &postDescription, &postID); err != nil {
			log.Println("Error while scanning %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while scanning"})
			return
		}

		post := map[string]interface{}{
			"user_id":          userID,
			"post_id":          postID,
			"user_name":        userName,
			"post_title":       postTitle,
			"post_description": postDescription,
		}

		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot iterate over the User."})
		return
	}
	response := GetUserPostsResponse{
		// response: posts,
		TotalCount: totalCount,
		Posts:      posts,

		// total:    totalCount,
	}
	c.JSON(http.StatusOK, response)

}

// func main() {
// 	// Define the connection string
// 	// Format: sqlserver://username:password@host:port?database=name
// 	connString := "sqlserver://sa:root@localhost:1433?database=blog_be"

// 	// Open a connection to the database
// 	db, err := sql.Open("sqlserver", connString)
// 	if err != nil {
// 		log.Fatalf("error opening connection: %v", err)
// 	}
// 	defer db.Close()

// 	// Test the connection
// 	err = db.Ping()
// 	if err != nil {
// 		log.Fatalf("error pinging database: %v", err)
// 	}
// 	fmt.Println("Successfully connected to the database")

// 	// Example query
// 	rows, err := db.Query("SELECT id, title FROM posts")
// 	if err != nil {
// 		log.Fatalf("error executing query: %v", err)
// 	}
// 	defer rows.Close()

// 	// Iterate over the results
// 	for rows.Next() {
// 		var id int
// 		var name string
// 		if err := rows.Scan(&id, &name); err != nil {
// 			log.Fatalf("error scanning row: %v", err)
// 		}
// 		fmt.Printf("ID: %d, Name: %s\n", id, name)
// 	}

// 	// Check for errors from iterating over rows
// 	if err := rows.Err(); err != nil {
// 		log.Fatalf("error iterating over rows: %v", err)
// 	}

// }
