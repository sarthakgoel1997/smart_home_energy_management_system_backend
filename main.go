package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"shems/users"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

func main() {
	router := mux.NewRouter()

	// MySQL database configuration
	db, err := sql.Open("mysql", "root:cricket97@tcp(localhost:3306)/Project")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()

	// POST API endpoint to login
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		users.LoginUser(ctx, w, r, db)
	})

	// POST API endpoint to register
	router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		users.RegisterUser(ctx, w, r, db, redisClient)
	})

	// GET API endpoint to fetch dashboard details
	router.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		users.GetDashboardData(ctx, w, r, db)
	})

	// GET API endpoint to fetch enrolled devices
	router.HandleFunc("/dashboard/getEnrolledDevices", func(w http.ResponseWriter, r *http.Request) {
		users.GetEnrolledDevices(ctx, w, r, db)
	})

	// POST API endpoint to add enrolled device
	router.HandleFunc("/dashboard/addEnrolledDevice", func(w http.ResponseWriter, r *http.Request) {
		users.AddEnrolledDevice(ctx, w, r, db, redisClient)
	})

	// PUT API endpoint to update enrolled device
	router.HandleFunc("/dashboard/updateEnrolledDevice", func(w http.ResponseWriter, r *http.Request) {
		users.UpdateEnrolledDevice(ctx, w, r, db, redisClient)
	})

	// DELETE API endpoint to delete enrolled device
	router.HandleFunc("/dashboard/deleteEnrolledDevice", func(w http.ResponseWriter, r *http.Request) {
		users.DeleteEnrolledDevice(ctx, w, r, db, redisClient)
	})

	// GET API endpoint to fetch service locations
	router.HandleFunc("/dashboard/getServiceLocations", func(w http.ResponseWriter, r *http.Request) {
		users.GetServiceLocations(ctx, w, r, db)
	})

	// POST API endpoint to add service location
	router.HandleFunc("/dashboard/addServiceLocation", func(w http.ResponseWriter, r *http.Request) {
		users.AddServiceLocation(ctx, w, r, db, redisClient)
	})

	// PUT API endpoint to update service location
	router.HandleFunc("/dashboard/updateServiceLocation", func(w http.ResponseWriter, r *http.Request) {
		users.UpdateServiceLocation(ctx, w, r, db, redisClient)
	})

	// DELETE API endpoint to delete service location
	router.HandleFunc("/dashboard/deleteServiceLocation", func(w http.ResponseWriter, r *http.Request) {
		users.DeleteServiceLocation(ctx, w, r, db, redisClient)
	})

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	port := 8000
	fmt.Printf("Server is running on :%d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler))
}
