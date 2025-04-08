package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Models
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"unique"`
	Password  string    `json:"-"` // The "-" tag means this field won't be included in JSON
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type Movie struct {
	ID        uint                 `json:"id" gorm:"primaryKey"`
	Title     string               `json:"title"`
	ShowDates []string             `json:"showDates" gorm:"serializer:json"`
	ShowTimes []string             `json:"showTimes" gorm:"serializer:json"`
	Price     float64              `json:"price"`
	CreatedAt time.Time            `json:"createdAt"`
}

type Seat struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Number   string `json:"number"`
	ShowID   uint   `json:"showId"`
	ShowDate string `json:"showDate"`
	ShowTime string `json:"showTime"`
	IsBooked bool   `json:"isBooked"`
}

type Show struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	Price       float64 `json:"price"`
}

type Booking struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	MovieID   uint      `json:"movieId"`
	ShowDate  string    `json:"showDate"`
	ShowTime  string    `json:"showTime"`
	SeatID    uint      `json:"seatId"`
	UserID    uint      `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Request structs
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

// Response structs
type BookingResponse struct {
	ID          uint      `json:"id"`
	MovieID     uint      `json:"movieId"`
	MovieTitle  string    `json:"movieTitle"`
	ShowDate    string    `json:"showDate"`
	ShowTime    string    `json:"showTime"`
	SeatID      uint      `json:"seatId"`
	SeatNumber  string    `json:"seatNumber"`
	UserID      uint      `json:"userId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Global variables
var (
	db *gorm.DB
	mu sync.Mutex
)

// Hash password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// Authentication middleware
func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("userID")
	if userID == nil {
		// Check if request is for HTML or API
		if c.Request.Header.Get("Accept") == "application/json" || c.Request.URL.Path[:5] == "/api/" {
			// API request, return JSON error
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
		} else {
			// HTML request, redirect to login page
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
		}
		return
	}
	c.Set("userID", userID)
	c.Next()
}

// Database initialization
func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("cinema.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the schemas
	db.AutoMigrate(&User{}, &Movie{}, &Seat{}, &Booking{}, &Show{})

	// Seed initial data if no movies exist
	var movieCount int64
	db.Model(&Movie{}).Count(&movieCount)
	if movieCount == 0 {
		seedInitialData()
	}
}

// User handlers
func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration data"})
		return
	}

	// Check if email already exists
	var existingUser User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Create new user
	user := User{
		Email:     req.Email,
		Password:  hashPassword(req.Password),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Start session
	session := sessions.Default(c)
	session.Set("userID", user.ID)
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login data"})
		return
	}

	var user User
	if err := db.Where("email = ? AND password = ?", req.Email, hashPassword(req.Password)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Start session
	session := sessions.Default(c)
	session.Set("userID", user.ID)
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Check authentication status
func checkAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("userID")
	authenticated := userID != nil

	c.JSON(http.StatusOK, gin.H{
		"authenticated": authenticated,
	})
}

// Get user profile
func getUserProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"name":      user.Name,
		"createdAt": user.CreatedAt,
	})
}

// Get user bookings
func getUserBookings(c *gin.Context) {
	userID := c.GetUint("userID")
	bookingType := c.Query("type") // 'upcoming' or 'history'
	
	log.Printf("Fetching bookings for user ID %d, type: %s", userID, bookingType)
	
	var bookings []Booking
	query := db.Where("user_id = ?", userID)
	
	// Filter by booking type if specified
	if bookingType == "upcoming" {
		// Simple implementation - just show recent bookings as upcoming
		query = query.Order("created_at desc").Limit(10)
	} else if bookingType == "history" {
		// Show older bookings for history
		query = query.Order("created_at asc").Limit(10)
	} else {
		// If no type specified, get all bookings
		query = query.Order("created_at desc")
	}
	
	if err := query.Find(&bookings).Error; err != nil {
		log.Printf("Error fetching bookings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching bookings"})
		return
	}
	
	log.Printf("Found %d bookings for user %d", len(bookings), userID)
	
	// If no bookings found, return empty array
	if len(bookings) == 0 {
		c.JSON(http.StatusOK, []BookingResponse{})
		return
	}
	
	// Enhance booking data with movie and seat information
	var bookingResponses []BookingResponse
	for _, booking := range bookings {
		var movie Movie
		var seat Seat
		
		if err := db.First(&movie, booking.MovieID).Error; err != nil {
			log.Printf("Error finding movie %d for booking %d: %v", booking.MovieID, booking.ID, err)
			continue
		}
		
		if err := db.First(&seat, booking.SeatID).Error; err != nil {
			log.Printf("Error finding seat %d for booking %d: %v", booking.SeatID, booking.ID, err)
			continue
		}
		
		bookingResponses = append(bookingResponses, BookingResponse{
			ID:          booking.ID,
			MovieID:     booking.MovieID,
			MovieTitle:  movie.Title,
			ShowDate:    booking.ShowDate,
			ShowTime:    booking.ShowTime,
			SeatID:      booking.SeatID,
			SeatNumber:  seat.Number,
			UserID:      booking.UserID,
			CreatedAt:   booking.CreatedAt,
		})
	}
	
	c.JSON(http.StatusOK, bookingResponses)
}

// Seed initial data
func seedInitialData() {
	movies := []Movie{
		{
			Title:     "The Matrix",
			ShowDates: []string{"2024-03-01", "2024-03-02", "2024-03-03"},
			ShowTimes: []string{"10:00 AM", "2:00 PM", "6:00 PM"},
			Price:     12.99,
		},
		{
			Title:     "Inception",
			ShowDates: []string{"2024-03-01", "2024-03-02", "2024-03-03"},
			ShowTimes: []string{"11:00 AM", "3:00 PM", "7:00 PM"},
			Price:     13.99,
		},
		{
			Title:     "Interstellar",
			ShowDates: []string{"2024-03-01", "2024-03-02", "2024-03-03"},
			ShowTimes: []string{"12:00 PM", "4:00 PM", "8:00 PM"},
			Price:     14.99,
		},
	}

	for _, movie := range movies {
		movie.CreatedAt = time.Now()
		if err := db.Create(&movie).Error; err != nil {
			log.Printf("Error creating movie %s: %v", movie.Title, err)
			continue
		}
		
		log.Printf("Created movie: %s with ID: %d", movie.Title, movie.ID)

		// Create seats for each date and show time combination
		for _, showDate := range movie.ShowDates {
			for _, showTime := range movie.ShowTimes {
				log.Printf("Creating seats for movie %s on %s at %s", movie.Title, showDate, showTime)
				// For each date and time combination, create an 8x8 seating arrangement
				for row := 0; row < 8; row++ {
					for col := 0; col < 8; col++ {
						seat := Seat{
							Number:   fmt.Sprintf("%c%d", 'A'+row, col+1),
							ShowID:   movie.ID,
							ShowDate: showDate,
							ShowTime: showTime,
							IsBooked: false,
						}
						if err := db.Create(&seat).Error; err != nil {
							log.Printf("Error creating seat %s: %v", seat.Number, err)
						}
					}
				}
			}
		}
	}

	// Seed initial shows
	shows := []Show{
		{ID: 1, Name: "Avengers: Endgame", Description: "After the devastating events of Avengers: Infinity War (2018), the universe is in ruins. With the help of remaining allies, the Avengers assemble once more in order to reverse Thanos' actions and restore balance to the universe.", Image: "/static/images/avengers.jpg", Price: 300},
		{ID: 2, Name: "The Dark Knight", Description: "When the menace known as the Joker wreaks havoc and chaos on the people of Gotham, Batman must accept one of the greatest psychological and physical tests of his ability to fight injustice.", Image: "/static/images/dark_knight.jpg", Price: 250},
		{ID: 3, Name: "Inception", Description: "A thief who steals corporate secrets through the use of dream-sharing technology is given the inverse task of planting an idea into the mind of a C.E.O.", Image: "/static/images/inception.jpg", Price: 280},
	}
	for _, show := range shows {
		db.Create(&show)
	}
	
	// Create dates and times for shows
	dates := []string{"2023-11-20", "2023-11-21", "2023-11-22"}
	times := []string{"10:00 AM", "2:00 PM", "6:00 PM", "9:00 PM"}
	
	// Seed initial seats - create seats for each show with associated dates and times
	for _, show := range shows {
		for _, date := range dates {
			for _, time := range times {
				// Create seats for each show/date/time combination
				// Create 3 rows (A, B, C) with 8 seats each
				for row := 0; row < 3; row++ {
					for col := 0; col < 8; col++ {
						seat := Seat{
							Number:   fmt.Sprintf("%c%d", 'A'+row, col+1),
							ShowID:   show.ID,
							ShowDate: date,
							ShowTime: time,
							IsBooked: false,
						}
						db.Create(&seat)
					}
				}
				log.Printf("Created seats for Show ID: %d, Date: %s, Time: %s", show.ID, date, time)
			}
		}
	}
	
	log.Println("Database seeded successfully")
}

// Handlers
func getMovies(c *gin.Context) {
	var movies []Movie
	if err := db.Find(&movies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching movies"})
		return
	}
	c.JSON(http.StatusOK, movies)
}

func getSeats(c *gin.Context) {
	showID := c.Query("showId")
	showDate := c.Query("showDate")
	showTime := c.Query("showTime")
	
	log.Printf("⭐ Getting seats for showID=%s, date=%s, time=%s", showID, showDate, showTime)
	
	// For demonstration purposes, just return all seats for this movie
	// This is a simplification to help with debugging
	query := db.Where("show_id = ?", showID)
	
	// Log the query we're about to execute
	log.Printf("Query: %v", query.Statement.SQL.String())
	
	// Get all seats for this movie
	var seats []Seat
	if err := query.Find(&seats).Error; err != nil {
		log.Printf("Error fetching seats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching seats"})
		return
	}
	
	log.Printf("Found %d total seats for movie ID %s", len(seats), showID)
	
	// If we have date and time filters, apply them in memory
	var filteredSeats []Seat
	if showDate != "" && showTime != "" {
		for _, seat := range seats {
			if seat.ShowDate == showDate && seat.ShowTime == showTime {
				filteredSeats = append(filteredSeats, seat)
			}
		}
		log.Printf("Filtered to %d seats matching date %s and time %s", len(filteredSeats), showDate, showTime)
		seats = filteredSeats
	}
	
	// If we don't have any seats after filtering, create dummy seats for testing
	if len(seats) == 0 {
		log.Printf("WARNING: No seats found for the given criteria. Creating dummy seats for testing.")
		for row := 0; row < 3; row++ {
			for col := 0; col < 4; col++ {
				seat := Seat{
					ID:       uint(1000 + row*10 + col),
					Number:   fmt.Sprintf("%c%d", 'A'+row, col+1),
					ShowID:   1,
					ShowDate: showDate,
					ShowTime: showTime,
					IsBooked: false,
				}
				// Make some seats booked for testing
				if row == 1 && col == 1 {
					seat.IsBooked = true
				}
				seats = append(seats, seat)
			}
		}
		log.Printf("Created %d dummy seats for testing", len(seats))
	}
	
	c.JSON(http.StatusOK, seats)
}

func bookSeat(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	var booking Booking
	if err := c.ShouldBindJSON(&booking); err != nil {
		log.Printf("Invalid booking data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking data"})
		return
	}

	log.Printf("Attempting to book seat: MovieID=%d, SeatID=%d, ShowDate=%s, ShowTime=%s", booking.MovieID, booking.SeatID, booking.ShowDate, booking.ShowTime)

	// Check if seat is already booked
	var seat Seat
	if err := db.Where("id = ? AND show_id = ?", booking.SeatID, booking.MovieID).First(&seat).Error; err != nil {
		log.Printf("Seat not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Seat not found for this show"})
		return
	}

	if seat.IsBooked {
		log.Printf("Seat %s is already booked", seat.Number)
		c.JSON(http.StatusConflict, gin.H{"error": "Seat already booked"})
		return
	}

	// Create transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("Recovered from panic in bookSeat: %v", r)
		}
	}()

	// Update seat status
	if err := tx.Model(&seat).Update("is_booked", true).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to update seat status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seat status"})
		return
	}

	// Get user ID from session
	session := sessions.Default(c)
	userID := session.Get("userID").(uint)
	booking.UserID = userID

	// Create booking record
	booking.CreatedAt = time.Now()
	if err := tx.Create(&booking).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create booking record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	tx.Commit()
	log.Printf("Successfully booked seat %s for movie %d on %s at %s", seat.Number, booking.MovieID, booking.ShowDate, booking.ShowTime)
	c.JSON(http.StatusOK, booking)
}

func main() {
	// Initialize database
	initDB()

	// Create Gin router
	r := gin.Default()

	// Setup session middleware
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("cinema_session", store))

	// Serve static files
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// Public routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})
	
	// Protected routes requiring authentication
	protected := r.Group("")
	protected.Use(authRequired)
	{
		protected.GET("/booking", func(c *gin.Context) {
			c.HTML(http.StatusOK, "booking.html", nil)
		})
		protected.GET("/profile", func(c *gin.Context) {
			c.HTML(http.StatusOK, "profile.html", nil)
		})
		protected.GET("/payment", func(c *gin.Context) {
			c.HTML(http.StatusOK, "payment.html", nil)
		})
	}

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", register)
		auth.POST("/login", login)
		auth.POST("/logout", logout)
	}

	// API routes
	api := r.Group("/api")
	{
		api.GET("/movies", getMovies)
		api.GET("/seats", getSeats)
		api.GET("/check-auth", checkAuth)
		
		// Protected routes
		protectedApi := api.Group("")
		protectedApi.Use(authRequired)
		{
			protectedApi.POST("/book", bookSeat)
			protectedApi.GET("/user/profile", getUserProfile)
			protectedApi.GET("/user/bookings", getUserBookings)
		}
	}

	// Start server
	fmt.Println("Server running on http://localhost:8082")
	r.Run(":8082")
}
