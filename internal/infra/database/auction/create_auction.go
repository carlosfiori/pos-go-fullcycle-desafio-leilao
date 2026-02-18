package auction

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/configuration/logger"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/entity/auction_entity"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/internal_error"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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
	auctionInterval time.Duration
	cancelFunc      context.CancelFunc
}

func NewAuctionRepository(ctx context.Context, database *mongo.Database) *AuctionRepository {
	auctionInterval := getAuctionInterval()
	checkInterval := getAuctionCloseCheckInterval()

	ctxWithCancel, cancel := context.WithCancel(ctx)

	ar := &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: auctionInterval,
		cancelFunc:      cancel,
	}

	go ar.monitorAuctionExpiration(ctxWithCancel, checkInterval)

	return ar
}

func (ar *AuctionRepository) monitorAuctionExpiration(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Auction expiration monitor stopped")
			return
		case <-ticker.C:
			ar.closeExpiredAuctions(ctx)
		}
	}
}

func (ar *AuctionRepository) closeExpiredAuctions(ctx context.Context) {
	expirationThreshold := time.Now().Add(-ar.auctionInterval).Unix()

	filter := bson.M{
		"status":    auction_entity.Active,
		"timestamp": bson.M{"$lte": expirationThreshold},
	}
	update := bson.M{
		"$set": bson.M{"status": auction_entity.Completed},
	}

	result, err := ar.Collection.UpdateMany(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to close expired auctions", err)
		return
	}

	if result.ModifiedCount > 0 {
		logger.Info(fmt.Sprintf("Closed %d expired auction(s)", result.ModifiedCount))
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

	return nil
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5
	}
	return duration
}

func getAuctionCloseCheckInterval() time.Duration {
	checkInterval := os.Getenv("AUCTION_CLOSE_CHECK_INTERVAL")
	duration, err := time.ParseDuration(checkInterval)
	if err != nil {
		return time.Second * 10
	}
	return duration
}
