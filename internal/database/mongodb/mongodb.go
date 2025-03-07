package mongodb

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	_ database.Database = (*MongoDB)(nil)
	_ database.Admin    = (*MongoDB)(nil)
)

const (
	// DefaultPort is the default port for MongoDB
	DefaultPort = 27017
	// DefaultUser is the default user for MongoDB
	DefaultUser = "mongodb"
	// DefaultPass is the default password for MongoDB
	DefaultPass = "mongodb"
	// DefaultName is the default database name for MongoDB
	DefaultName = "admin"
)

// MongoDB is a MongoDB database instance
type MongoDB struct {
	containerID string
	cfg         config
}

// New creates a new MongoDB database instance controller
func New(options ...Option) (*MongoDB, error) {
	// create MongoDB with default values
	m := &MongoDB{cfg: config{
		pass:    DefaultPass,
		user:    DefaultUser,
		name:    DefaultName,
		port:    DefaultPort,
		version: "7.0", // Default to latest version
	}}

	for _, o := range options {
		if err := o(&m.cfg); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// CreateDB creates a new database with given migrations and fixtures
func (m *MongoDB) CreateDB(ctx context.Context, req *database.CreateDBRequest) (*database.CreateDBResponse, error) {
	// Connect to MongoDB
	client, err := m.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB failed: %w", err)
	}
	defer client.Disconnect(ctx)

	// Generate a random name for the new database
	dbName := fmt.Sprintf("dbctl_%d", time.Now().UnixNano())

	// Create a new MongoDB instance with the new database name
	newDB, _ := New(WithHost(m.cfg.user, m.cfg.pass, dbName, m.cfg.port))
	newURI := newDB.URI()

	// Create the database by simply accessing it (MongoDB creates databases on first use)
	db := client.Database(dbName)

	// Apply migrations if they exist
	if len(req.Migrations) > 0 {
		logger.Info("Applying migrations to MongoDB...")
		if err := applyMigrationsFromDir(ctx, db, req.Migrations); err != nil {
			return nil, fmt.Errorf("applying migrations failed: %w", err)
		}
	}

	// Apply fixtures if they exist
	if len(req.Fixtures) > 0 {
		logger.Info("Applying fixtures to MongoDB...")
		if err := applyFixturesFromDir(ctx, db, req.Fixtures); err != nil {
			return nil, fmt.Errorf("applying fixtures failed: %w", err)
		}
	}

	// Make sure we return localhost instead of host.docker.internal
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		newURI = strings.ReplaceAll(newURI, "host.docker.internal", "localhost")
	}

	return &database.CreateDBResponse{URI: newURI}, nil
}

// RemoveDB removes a database from MongoDB by given URI
func (m *MongoDB) RemoveDB(ctx context.Context, uri string) error {
	// Parse the URI to get database name
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// Get database name - take the path and remove leading slash
	dbName := strings.TrimPrefix(u.Path, "/")

	// Connect to MongoDB
	client, err := m.connect(ctx)
	if err != nil {
		return fmt.Errorf("connect to MongoDB failed: %w", err)
	}
	defer client.Disconnect(ctx)

	// Drop the database
	if err := client.Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("drop database failed: %w", err)
	}

	return nil
}

// Start starts a MongoDB database
func (m *MongoDB) Start(ctx context.Context, detach bool) error {
	logger.Info(fmt.Sprintf("Starting MongoDB version %s on port %d ...", m.cfg.version, m.cfg.port))

	closeFunc, err := m.startUsingDocker(ctx, 20*time.Second)
	if err != nil {
		return err
	}

	logger.Info("MongoDB is up and running")

	// Run migrations if they exist
	if err := m.runMigrations(ctx); err != nil {
		return err
	}

	// Apply fixtures if they exist
	if err := m.applyFixtures(ctx); err != nil {
		return err
	}

	// Print connection URL
	logger.Info(fmt.Sprintf("Database URI is: %q", m.URI()))

	var mongoExpressCloseFunc database.CloseFunc
	if m.cfg.withUI {
		mongoExpressCloseFunc, err = m.runUI(ctx)
		if err != nil {
			_ = closeFunc(ctx)
			return err
		}
	}

	// Detach and stop CLI if asked
	if detach {
		return nil
	}

	<-ctx.Done()
	logger.Info("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if mongoExpressCloseFunc != nil {
		if err := mongoExpressCloseFunc(shutdownCtx); err != nil {
			return err
		}
	}

	return closeFunc(shutdownCtx)
}

// Stop stops a MongoDB database
func (m *MongoDB) Stop(ctx context.Context) error {
	return container.TerminateByID(ctx, m.containerID)
}

// WaitForStart waits for MongoDB to start
func (m *MongoDB) WaitForStart(ctx context.Context, timeout time.Duration) error {
	logger.Info("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for range ticker.C {
		client, err := m.connect(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return err
			}
		} else {
			_ = client.Disconnect(ctx)
			return nil
		}
	}
	return nil
}

// Instances returns a list of MongoDB instances
func Instances(ctx context.Context) ([]database.Info, error) {
	l, err := container.List(ctx, map[string]string{container.LabelType: database.LabelMongoDB})
	if err != nil {
		return nil, err
	}

	out := make([]database.Info, 0, len(l))
	for _, c := range l {
		out = append(out, database.Info{
			ID:     c.ID,
			Type:   c.Name,
			Status: database.Running,
		})
	}
	return out, nil
}

func (m *MongoDB) startUsingDocker(ctx context.Context, timeout time.Duration) (database.CloseFunc, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	port := strconv.Itoa(int(m.cfg.port))
	req := container.CreateRequest{
		Image: getMongoDBImage(m.cfg.version),
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": m.cfg.user,
			"MONGO_INITDB_ROOT_PASSWORD": m.cfg.pass,
		},
		ExposedPorts: []string{fmt.Sprintf("%s:27017/tcp", port)},
		Name:         fmt.Sprintf("dbctl_mongo_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelMongoDB},
	}

	if m.cfg.label != "" {
		req.Labels[container.LabelCustom] = m.cfg.label
	}

	mongo, err := container.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	m.containerID = mongo.ID

	closeFunc := func(ctx context.Context) error {
		return mongo.Terminate(ctx)
	}

	return closeFunc, m.WaitForStart(ctx, timeout)
}

func (m *MongoDB) runUI(ctx context.Context) (database.CloseFunc, error) {
	logger.Info("Starting MongoDB UI using mongo-express")

	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	// Choose a port for mongo-express
	expressPort := "8081"

	mongoExpress, err := container.Run(ctx, container.CreateRequest{
		Image: "mongo-express:latest",
		Env: map[string]string{
			"ME_CONFIG_BASICAUTH_ENABLED": "false",
			"ME_CONFIG_MONGODB_URL":       strings.ReplaceAll(m.URI(), "localhost", "host.docker.internal"),
		},
		ExposedPorts: []string{fmt.Sprintf("%s:8081", expressPort)},
		Name:         fmt.Sprintf("dbctl_mongo_express_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelMongoExpress},
	})
	if err != nil {
		return nil, err
	}

	// Log UI URL
	logger.Info("Database UI is running on: http://localhost:" + expressPort)

	closeFunc := func(ctx context.Context) error {
		return mongoExpress.Terminate(ctx)
	}

	return closeFunc, nil
}

// URI returns the MongoDB connection URI
func (m *MongoDB) URI() string {
	addr := "localhost"
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		addr = "host.docker.internal"
	}

	host := net.JoinHostPort(addr, strconv.Itoa(int(m.cfg.port)))

	// Create the connection URI
	uri := fmt.Sprintf("mongodb://%s:%s@%s/%s", url.QueryEscape(m.cfg.user), url.QueryEscape(m.cfg.pass), host, m.cfg.name)
	return uri
}

// Helper function to connect to MongoDB
func (m *MongoDB) connect(ctx context.Context) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(m.URI()))
	if err != nil {
		return nil, err
	}

	// Ping to confirm connection is successful
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}

// Run migrations on the MongoDB database
func (m *MongoDB) runMigrations(ctx context.Context) error {
	if len(m.cfg.migrationsFiles) == 0 {
		return nil
	}

	logger.Info("Applying migrations...")

	// Connect to the database
	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	db := client.Database(m.cfg.name)

	// Apply migrations
	for _, file := range m.cfg.migrationsFiles {
		if err := applyJSScript(ctx, db, file); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}
	}

	return nil
}

// Apply fixtures to the MongoDB database
func (m *MongoDB) applyFixtures(ctx context.Context) error {
	if len(m.cfg.fixtureFiles) == 0 {
		return nil
	}

	logger.Info("Applying fixtures...")

	// Connect to the database
	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	db := client.Database(m.cfg.name)

	// Apply fixtures
	for _, file := range m.cfg.fixtureFiles {
		if err := applyJSScript(ctx, db, file); err != nil {
			return fmt.Errorf("failed to apply fixture %s: %w", file, err)
		}
	}

	return nil
}

// Apply migrations from a directory
func applyMigrationsFromDir(ctx context.Context, db *mongo.Database, dir string) error {
	if dir == "" {
		return nil
	}

	files, err := getFiles(dir)
	if err != nil {
		return fmt.Errorf("read migrations failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Applying %d migration files...", len(files)))

	for _, file := range files {
		logger.Info(fmt.Sprintf("Applying migration: %s", filepath.Base(file)))
		if err := applyJSScript(ctx, db, file); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}
	}

	return nil
}

// Apply fixtures from a directory
func applyFixturesFromDir(ctx context.Context, db *mongo.Database, dir string) error {
	if dir == "" {
		return nil
	}

	files, err := getFiles(dir)
	if err != nil {
		return fmt.Errorf("read fixtures failed: %w", err)
	}

	logger.Info(fmt.Sprintf("Applying %d fixture files...", len(files)))

	for _, file := range files {
		logger.Info(fmt.Sprintf("Applying fixture: %s", filepath.Base(file)))
		if err := applyJSScript(ctx, db, file); err != nil {
			return fmt.Errorf("failed to apply fixture %s: %w", file, err)
		}
	}

	return nil
}

// Apply a JavaScript script to the MongoDB database
// This is used for both migrations and fixtures
func applyJSScript(ctx context.Context, db *mongo.Database, scriptPath string) error {
	// Read the script file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// For MongoDB, we need to run the JavaScript through the mongo shell
	// We'll use the same container to run the mongo shell command

	// Get file extension
	ext := strings.ToLower(filepath.Ext(scriptPath))

	// Handle different types of scripts
	switch ext {
	case ".js":
		// JavaScript file for MongoDB shell
		// We need to use docker exec to run the mongo shell command
		containers, err := container.List(ctx, map[string]string{container.LabelType: database.LabelMongoDB})
		if err != nil || len(containers) == 0 {
			return fmt.Errorf("failed to find MongoDB container: %w", err)
		}

		// We'll use the DefaultUser and DefaultPass
		// since those are what we set up the container with
		cmd := []string{
			"mongosh",
			"--username", DefaultUser,
			"--password", DefaultPass,
			"--authenticationDatabase", "admin",
			db.Name(),
			"--eval", string(content),
		}

		// Run the command in the container
		output, err := container.RunExec(ctx, containers[0].ID, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute script: %w", err)
		}

		// Check if the output contains error messages
		if strings.Contains(strings.ToLower(output), "error") ||
			strings.Contains(strings.ToLower(output), "exception") {
			logger.Error(fmt.Sprintf("Error in script %s: %s", scriptPath, output))
			return fmt.Errorf("script execution error: %s", output)
		}

		logger.Info(fmt.Sprintf("Applied script %s successfully", filepath.Base(scriptPath)))

	case ".json":
		// JSON file for MongoDB - import into a collection based on filename
		collName := filepath.Base(scriptPath)
		collName = strings.TrimSuffix(collName, ext)

		// Parse JSON file content
		var documents []interface{}
		if err := json.Unmarshal(content, &documents); err != nil {
			return fmt.Errorf("failed to parse JSON file: %w", err)
		}

		// Insert documents
		if len(documents) > 0 {
			_, err := db.Collection(collName).InsertMany(ctx, documents)
			if err != nil {
				return fmt.Errorf("failed to insert documents: %w", err)
			}

			logger.Info(fmt.Sprintf("Imported %d documents into collection %s", len(documents), collName))
		}

	default:
		return fmt.Errorf("unsupported script file type: %s", ext)
	}

	return nil
}
