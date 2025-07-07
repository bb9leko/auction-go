package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"strconv"
	"sync"
	"time"

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
	Collection *mongo.Collection
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// Função para calcular o tempo do leilão via variável de ambiente
func (ar *AuctionRepository) GetAuctionDuration() time.Duration {
	durationStr := os.Getenv("AUCTION_DURATION")
	if durationStr == "" {
		durationStr = "10"
	}
	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		logger.Error("Invalid AUCTION_DURATION, using default 10 seconds", err)
		duration = 10
	}
	return time.Duration(duration) * time.Second
}

// Função para calcular o intervalo de checagem via variável de ambiente
func (ar *AuctionRepository) GetAuctionInterval() time.Duration {
	intervalStr := os.Getenv("AUCTION_INTERVAL")
	if intervalStr == "" {
		intervalStr = "2"
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		logger.Error("Invalid AUCTION_INTERVAL, using default 2 seconds", err)
		interval = 2
	}
	return time.Duration(interval) * time.Second
}

// Goroutine para fechamento automático
func (ar *AuctionRepository) startAuctionAutoCloser() {
	ar.wg.Add(1)
	go func() {
		defer ar.wg.Done()
		ticker := time.NewTicker(ar.GetAuctionInterval())
		defer ticker.Stop()
		logger.Info("Auction auto-closer goroutine started")
		for {
			select {
			case <-ar.ctx.Done():
				logger.Info("Auction auto-closer stopped")
				return
			case <-ticker.C:
				ar.closeExpiredAuctions()
			}
		}
	}()
}

// Lógica de update automático de status
func (ar *AuctionRepository) closeExpiredAuctions() {
	auctionDuration := ar.GetAuctionDuration()
	now := time.Now()
	expirationTime := now.Add(-auctionDuration)
	filter := bson.M{
		"status":    auction_entity.Active,
		"timestamp": bson.M{"$lt": expirationTime.Unix()},
	}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}
	result, err := ar.Collection.UpdateMany(ar.ctx, filter, update)
	if err != nil {
		logger.Error("Error closing expired auctions", err)
		return
	}
	if result.ModifiedCount > 0 {
		logger.Info(fmt.Sprintf("Closed %d expired auctions", result.ModifiedCount))
	}
}

// Graceful shutdown
func (ar *AuctionRepository) Close() {
	ar.cancel()
	ar.wg.Wait()
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())
	ar := &AuctionRepository{
		Collection: database.Collection("auctions"),
		ctx:        ctx,
		cancel:     cancel,
	}
	ar.startAuctionAutoCloser()
	return ar
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
