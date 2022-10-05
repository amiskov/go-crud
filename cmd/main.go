package main

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crud/pkg/logger"
	"crud/pkg/middleware"
	"crud/pkg/post"
	"crud/pkg/sessions"
	"crud/pkg/user"
	"crud/pkg/user/api"
)

type EnvConfig map[string]string

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var cfg EnvConfig = readDotenv()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("pgx", "postgresql://localhost/"+cfg["POSTGRES_DB"]+"?sslmode=disable")
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("unable to reach PostgreSQL: %v", err)
	}

	redisConn, err := redis.DialURL(cfg["REDIS_ADDR"])
	if err != nil {
		log.Fatalf("main: can't connect to Redis")
	}

	mongoCtx, mongoCtxCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer mongoCtxCancel()
	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg["MONGODB_URI"]))
	if err != nil {
		log.Fatalln("main: can't connect to MongoDB,", err)
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		log.Fatalln("main: unable to connect to MongoDB,", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(mongoCtx); err != nil {
			log.Fatalln("main: failed disconnecting from MongoDB, ", err)
		}
	}()

	postsDB := mongoClient.Database("crud").Collection("posts")
	postsRepo := post.NewPostRepo(postsDB)
	usersRepo := user.NewUserRepo(db)
	sessionManager := sessions.NewSessionManager(cfg["SECRET_KEY"], redisConn)
	postHandler := post.NewPostHandler(postsRepo)
	userHandler := api.NewUserHanler(usersRepo, sessionManager)

	r := mux.NewRouter()

	// Generate fake content to have better UI experience
	// seed(usersRepo, postsRepo)

	api := r.PathPrefix("/api").Subrouter()

	// Posts
	api.HandleFunc("/posts/", postHandler.List).Methods("GET")
	api.HandleFunc("/posts", postHandler.Add).Methods("POST")
	api.HandleFunc("/post/{post_id}", postHandler.Get).Methods("GET")
	api.HandleFunc("/post/{post_id}", postHandler.Delete).Methods("DELETE")
	// GET был сделан автором оригинального фронта, я пока не добрался форкнуть и поправить.
	api.HandleFunc("/post/{post_id}/upvote", postHandler.Upvote).Methods("GET")
	api.HandleFunc("/post/{post_id}/downvote", postHandler.Downvote).Methods("GET")
	api.HandleFunc("/post/{post_id}/unvote", postHandler.Unvote).Methods("GET")
	api.HandleFunc("/user/{username}", postHandler.GetByUser).Methods("GET")
	api.HandleFunc("/posts/{category}", postHandler.GetCategory).Methods("GET")

	// Comments
	api.HandleFunc("/post/{post_id}", postHandler.AddComment).Methods("POST")
	api.HandleFunc("/post/{post_id}/{comment_id}", postHandler.DeleteComment).Methods("DELETE")

	// User
	api.HandleFunc("/register", userHandler.Register).Methods("POST")
	api.HandleFunc("/login", userHandler.LogIn).Methods("POST")

	auth := middleware.NewAuthMiddleware(sessionManager, usersRepo)
	r.Use(auth.Middleware)

	logMiddleware := middleware.NewLoggingMiddleware(logger.Run(cfg["LOG_LEVEL"]))
	r.Use(logMiddleware.SetupTracing)
	r.Use(logMiddleware.SetupLogging)
	r.Use(logMiddleware.AccessLog)

	// Template path is relative to the project root for running with Makefile
	spa := spaHandler{staticPath: "template", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	log.Println("Serving at http://localhost:8080/")
	log.Fatalln(http.ListenAndServe(":8080", r))
}

func readDotenv() EnvConfig {
	env, err := godotenv.Read()
	if err != nil {
		log.Fatal("failed reading .env file:", err)
	}
	return env
}
