package auction

import (
	"context"
	"github.com/Alef-Daniel/challenge-go-auctions/configuration/logger"
	"github.com/Alef-Daniel/challenge-go-auctions/internal/entity/auction_entity"
	"github.com/Alef-Daniel/challenge-go-auctions/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const defaultAuctionDuration = 5 * time.Minute

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionDuration time.Duration
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionDuration: GetAuctionDuration(),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	endTime := auctionEntity.Timestamp.Add(ar.auctionDuration)
	go ar.closeAuctionAt(auctionEntity.Id, endTime)

	return nil
}

func (ar *AuctionRepository) closeAuctionAt(auctionId string, endTime time.Time) {
	timer := time.NewTimer(time.Until(endTime))
	defer timer.Stop()
	<-timer.C

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": auctionId, "status": auction_entity.Active}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	if _, err := ar.Collection.UpdateOne(ctx, filter, update); err != nil {
		logger.Error("Error trying to close auction", err)
		return
	}

	logger.Info("Auction closed automatically", zap.String("auction_id", auctionId))
}

func GetAuctionDuration() time.Duration {
	for _, key := range []string{"AUCTION_DURATION", "AUCTION_INTERVAL"} {
		if duration, err := time.ParseDuration(os.Getenv(key)); err == nil && duration > 0 {
			return duration
		}
	}

	return defaultAuctionDuration
}
