package storage

import (
	"context"
	"encoding/json"
	"time"

	"dungeon-shop/internal/models"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Storage struct {
	redisClient *redis.Client
	mongoClient *mongo.Client
	ctx         context.Context
}

func NewStorage(redisAddr, mongoURI string) (*Storage, error) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &Storage{
		redisClient: rdb,
		mongoClient: mongoClient,
		ctx:         ctx,
	}, nil
}

func (s *Storage) Close() error {
	if err := s.redisClient.Close(); err != nil {
		return err
	}
	return s.mongoClient.Disconnect(s.ctx)
}

func (s *Storage) SaveRoomState(roomID string, room *models.Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return err
	}
	return s.redisClient.Set(s.ctx, "room:"+roomID, data, 24*time.Hour).Err()
}

func (s *Storage) GetRoomState(roomID string) (*models.Room, error) {
	data, err := s.redisClient.Get(s.ctx, "room:"+roomID).Bytes()
	if err != nil {
		return nil, err
	}

	var room models.Room
	if err := json.Unmarshal(data, &room); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Storage) UpdateLeaderboard(playerID string, playerName string, assets int, won bool) error {
	pipe := s.redisClient.Pipeline()
	pipe.ZAdd(s.ctx, "leaderboard", redis.Z{
		Score:  float64(assets),
		Member: playerID + ":" + playerName,
	})

	if won {
		pipe.ZIncrBy(s.ctx, "wins:"+playerID, 1, "")
	}

	_, err := pipe.Exec(s.ctx)
	return err
}

func (s *Storage) GetLeaderboard(limit int) ([]redis.Z, error) {
	return s.redisClient.ZRevRangeWithScores(s.ctx, "leaderboard", 0, int64(limit)-1).Result()
}

func (s *Storage) SaveGameRecord(record *models.GameRecord) error {
	collection := s.mongoClient.Database("dungeon_shop").Collection("game_records")
	_, err := collection.InsertOne(s.ctx, record)
	return err
}

func (s *Storage) GetGameRecords(playerID string, limit int) ([]models.GameRecord, error) {
	collection := s.mongoClient.Database("dungeon_shop").Collection("game_records")

	filter := bson.M{"players.playerId": playerID}
	opts := options.Find().SetSort(bson.D{{Key: "endTime", Value: -1}}).SetLimit(int64(limit))

	cursor, err := collection.Find(s.ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var records []models.GameRecord
	if err := cursor.All(s.ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Storage) SavePlayerProfile(profile *models.PlayerProfile) error {
	collection := s.mongoClient.Database("dungeon_shop").Collection("player_profiles")

	filter := bson.M{"id": profile.ID}
	update := bson.M{
		"$set": profile,
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

func (s *Storage) GetPlayerProfile(playerID string) (*models.PlayerProfile, error) {
	collection := s.mongoClient.Database("dungeon_shop").Collection("player_profiles")

	var profile models.PlayerProfile
	err := collection.FindOne(s.ctx, bson.M{"id": playerID}).Decode(&profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *Storage) UpdatePlayerStats(playerID string, won bool, earnings int64) error {
	collection := s.mongoClient.Database("dungeon_shop").Collection("player_profiles")

	update := bson.M{
		"$inc": bson.M{
			"totalGames":   1,
			"totalEarnings": earnings,
		},
	}

	if won {
		update["$inc"].(bson.M)["wins"] = 1
	}

	_, err := collection.UpdateOne(s.ctx, bson.M{"id": playerID}, update)
	return err
}
