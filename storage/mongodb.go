// Package storage provides MongoDB storage implementation for the reaction system.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBConfig holds configuration for MongoDB connection.
type MongoDBConfig struct {
	// URI is the MongoDB connection URI.
	// Example: mongodb://localhost:27017
	URI string

	// Database is the database name.
	Database string

	// Collection is the collection name for reactions.
	Collection string

	// MaxPoolSize is the maximum connection pool size.
	MaxPoolSize uint64

	// MinPoolSize is the minimum connection pool size.
	MinPoolSize uint64

	// MaxConnIdleTime is the maximum connection idle time.
	MaxConnIdleTime time.Duration
}

// DefaultMongoDBConfig returns a default configuration.
func DefaultMongoDBConfig() MongoDBConfig {
	return MongoDBConfig{
		URI:             "mongodb://localhost:27017",
		Database:        "golikeit",
		Collection:      "reactions",
		MaxPoolSize:     100,
		MinPoolSize:     10,
		MaxConnIdleTime: 10 * time.Minute,
	}
}

// MongoDBStorage implements Repository using MongoDB.
type MongoDBStorage struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// MongoReaction represents a reaction document in MongoDB.
type MongoReaction struct {
	ID           string    `bson:"_id,omitempty"`
	UserID       string    `bson:"user_id"`
	EntityType   string    `bson:"entity_type"`
	EntityID     string    `bson:"entity_id"`
	ReactionType string    `bson:"reaction_type"`
	CreatedAt    time.Time `bson:"created_at"`
}

// NewMongoDBStorage creates a new MongoDB storage instance.
func NewMongoDBStorage(ctx context.Context, config MongoDBConfig) (*MongoDBStorage, error) {
	clientOpts := options.Client().ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetMaxConnIdleTime(config.MaxConnIdleTime)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(config.Database)
	coll := db.Collection(config.Collection)

	return &MongoDBStorage{
		client:     client,
		database:   db,
		collection: coll,
	}, nil
}

// InitSchema creates indexes for optimal query performance.
func (m *MongoDBStorage) InitSchema(ctx context.Context) error {
	// Create unique index on user_id + entity_type + entity_id
	userEntityIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "entity_type", Value: 1},
			{Key: "entity_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// Create index for entity queries
	entityIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "entity_type", Value: 1},
			{Key: "entity_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	}

	// Create index for user queries
	userIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	}

	// Create index for reaction type queries
	reactionTypeIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "reaction_type", Value: 1},
		},
	}

	_, err := m.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		userEntityIndex,
		entityIndex,
		userIndex,
		reactionTypeIndex,
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// AddReaction adds or replaces a reaction for a user.
func (m *MongoDBStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	filter := bson.M{
		"user_id":      userID,
		"entity_type":  target.EntityType,
		"entity_id":    target.EntityID,
	}

	// Check if reaction exists
	var existing MongoReaction
	err := m.collection.FindOne(ctx, filter).Decode(&existing)
	isReplacement := err == nil

	if isReplacement {
		// Update existing reaction
		update := bson.M{
			"$set": bson.M{
				"reaction_type": reactionType,
				"created_at":    time.Now().UTC(),
			},
		}
		_, err = m.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return false, fmt.Errorf("failed to update reaction: %w", err)
		}
		return true, nil
	}

	// Insert new reaction
	reaction := MongoReaction{
		UserID:       userID,
		EntityType:   target.EntityType,
		EntityID:     target.EntityID,
		ReactionType: reactionType,
		CreatedAt:    time.Now().UTC(),
	}
	_, err = m.collection.InsertOne(ctx, reaction)
	if err != nil {
		return false, fmt.Errorf("failed to insert reaction: %w", err)
	}

	return false, nil
}

// RemoveReaction removes a user's reaction.
func (m *MongoDBStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	filter := bson.M{
		"user_id":      userID,
		"entity_type":  target.EntityType,
		"entity_id":    target.EntityID,
	}

	result, err := m.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	if result.DeletedCount == 0 {
		return golikeit.ErrReactionNotFound
	}

	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (m *MongoDBStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	filter := bson.M{
		"user_id":      userID,
		"entity_type":  target.EntityType,
		"entity_id":    target.EntityID,
	}

	var reaction MongoReaction
	err := m.collection.FindOne(ctx, filter).Decode(&reaction)
	if err == mongo.ErrNoDocuments {
		return "", golikeit.ErrReactionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user reaction: %w", err)
	}

	return reaction.ReactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (m *MongoDBStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	filter := bson.M{
		"user_id":      userID,
		"entity_type":  target.EntityType,
		"entity_id":    target.EntityID,
	}

	count, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check user reaction: %w", err)
	}

	return count > 0, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (m *MongoDBStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"entity_type": target.EntityType,
			"entity_id":   target.EntityID,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$reaction_type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("failed to aggregate entity counts: %w", err)
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int64)
	var total int64

	for cursor.Next(ctx) {
		var result struct {
			ReactionType string `bson:"_id"`
			Count        int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return golikeit.EntityCounts{}, fmt.Errorf("failed to decode count: %w", err)
		}
		counts[result.ReactionType] = result.Count
		total += result.Count
	}

	if err := cursor.Err(); err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("error iterating counts: %w", err)
	}

	return golikeit.EntityCounts{
		Counts: counts,
		Total:  total,
	}, nil
}

// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
func (m *MongoDBStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	filter := bson.M{"user_id": userID}

	if filters.EntityType != "" {
		filter["entity_type"] = filters.EntityType
	}
	if filters.ReactionType != "" {
		filter["reaction_type"] = filters.ReactionType
	}
	if filters.Since != nil || filters.Until != nil {
		dateFilter := bson.M{}
		if filters.Since != nil {
			dateFilter["$gte"] = *filters.Since
		}
		if filters.Until != nil {
			dateFilter["$lte"] = *filters.Until
		}
		filter["created_at"] = dateFilter
	}

	// Get total count
	total, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated results
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(pag.Offset)).
		SetLimit(int64(pag.Limit))

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reactions: %w", err)
	}
	defer cursor.Close(ctx)

	var reactions []golikeit.UserReaction
	for cursor.Next(ctx) {
		var mr MongoReaction
		if err := cursor.Decode(&mr); err != nil {
			return nil, 0, fmt.Errorf("failed to decode reaction: %w", err)
		}
		reactions = append(reactions, golikeit.UserReaction{
			UserID:       mr.UserID,
			EntityType:   mr.EntityType,
			EntityID:     mr.EntityID,
			ReactionType: mr.ReactionType,
			CreatedAt:    mr.CreatedAt,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetUserReactionCounts returns aggregated counts per reaction type for a user.
func (m *MongoDBStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	filter := bson.M{"user_id": userID}
	if entityTypeFilter != "" {
		filter["entity_type"] = entityTypeFilter
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$reaction_type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate user reaction counts: %w", err)
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ReactionType string `bson:"_id"`
			Count        int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode count: %w", err)
		}
		counts[result.ReactionType] = result.Count
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counts: %w", err)
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user.
func (m *MongoDBStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	filter := bson.M{
		"user_id":       userID,
		"reaction_type": reactionType,
	}

	total, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(pag.Offset)).
		SetLimit(int64(pag.Limit))

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reactions: %w", err)
	}
	defer cursor.Close(ctx)

	var reactions []golikeit.UserReaction
	for cursor.Next(ctx) {
		var mr MongoReaction
		if err := cursor.Decode(&mr); err != nil {
			return nil, 0, fmt.Errorf("failed to decode reaction: %w", err)
		}
		reactions = append(reactions, golikeit.UserReaction{
			UserID:       mr.UserID,
			EntityType:   mr.EntityType,
			EntityID:     mr.EntityID,
			ReactionType: mr.ReactionType,
			CreatedAt:    mr.CreatedAt,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetEntityReactions retrieves all reactions on an entity with pagination.
func (m *MongoDBStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	filter := bson.M{
		"entity_type": target.EntityType,
		"entity_id":   target.EntityID,
	}

	total, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(pag.Offset)).
		SetLimit(int64(pag.Limit)).
		SetProjection(bson.M{
			"user_id":       1,
			"reaction_type": 1,
			"created_at":    1,
		})

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find reactions: %w", err)
	}
	defer cursor.Close(ctx)

	var reactions []golikeit.EntityReaction
	for cursor.Next(ctx) {
		var mr MongoReaction
		if err := cursor.Decode(&mr); err != nil {
			return nil, 0, fmt.Errorf("failed to decode reaction: %w", err)
		}
		reactions = append(reactions, golikeit.EntityReaction{
			UserID:       mr.UserID,
			ReactionType: mr.ReactionType,
			CreatedAt:    mr.CreatedAt,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetRecentReactions retrieves recent reactions on an entity.
func (m *MongoDBStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	filter := bson.M{
		"entity_type": target.EntityType,
		"entity_id":   target.EntityID,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetProjection(bson.M{
			"user_id":       1,
			"reaction_type": 1,
			"created_at":    1,
		})

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find recent reactions: %w", err)
	}
	defer cursor.Close(ctx)

	var reactions []golikeit.RecentUserReaction
	for cursor.Next(ctx) {
		var mr MongoReaction
		if err := cursor.Decode(&mr); err != nil {
			return nil, fmt.Errorf("failed to decode reaction: %w", err)
		}
		reactions = append(reactions, golikeit.RecentUserReaction{
			UserID:       mr.UserID,
			ReactionType: mr.ReactionType,
			Timestamp:    mr.CreatedAt,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, nil
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
func (m *MongoDBStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	filter := bson.M{
		"entity_type": target.EntityType,
		"entity_id":   target.EntityID,
	}

	opts := options.FindOne().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetProjection(bson.M{"created_at": 1})

	var result struct {
		CreatedAt time.Time `bson:"created_at"`
	}
	err := m.collection.FindOne(ctx, filter, opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	return &result.CreatedAt, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (m *MongoDBStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := m.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := m.GetRecentReactions(ctx, target, maxRecentUsers*10)
		if err != nil {
			return golikeit.EntityReactionDetail{}, err
		}

		// Group by reaction type
		for _, r := range recent {
			recentUsers[r.ReactionType] = append(recentUsers[r.ReactionType], r)
		}

		// Trim to maxRecentUsers per type
		for rt, users := range recentUsers {
			if len(users) > maxRecentUsers {
				recentUsers[rt] = users[:maxRecentUsers]
			}
		}
	}

	lastTime, err := m.GetLastReactionTime(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	return golikeit.EntityReactionDetail{
		EntityType:     target.EntityType,
		EntityID:       target.EntityID,
		TotalReactions: counts.Total,
		CountsByType:   counts.Counts,
		RecentUsers:    recentUsers,
		LastReaction:   lastTime,
	}, nil
}

// Close releases resources held by the storage.
func (m *MongoDBStorage) Close() error {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return m.client.Disconnect(ctx)
	}
	return nil
}
