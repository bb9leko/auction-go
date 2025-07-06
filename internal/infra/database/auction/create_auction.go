package auction

import (
	"context"
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

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())
	ar := &AuctionRepository{
		Collection: database.Collection("auctions"),
		ctx:        ctx,
		cancel:     cancel,
	}
	// Inicia a goroutine para fechamento automático
	ar.startAuctionAutoCloser()

	return ar
}

// GetAuctionDuration retorna a duração do leilão baseada na variável de ambiente
func (ar *AuctionRepository) GetAuctionDuration() time.Duration {
	durationStr := os.Getenv("AUCTION_DURATION")
	if durationStr == "" {
		durationStr = "300" // 5 minutos como padrão
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		logger.Error("Invalid AUCTION_DURATION, using default 300 seconds", err)
		duration = 300
	}

	return time.Duration(duration) * time.Second
}

// GetAuctionInterval retorna o intervalo de verificação baseado na variável de ambiente
func (ar *AuctionRepository) GetAuctionInterval() time.Duration {
	intervalStr := os.Getenv("AUCTION_INTERVAL")
	if intervalStr == "" {
		intervalStr = "60" // 1 minuto como padrão
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		logger.Error("Invalid AUCTION_INTERVAL, using default 60 seconds", err)
		interval = 60
	}

	return time.Duration(interval) * time.Second
}

// startAuctionAutoCloser inicia uma goroutine para fechamento automático de leilões
func (ar *AuctionRepository) startAuctionAutoCloser() {
	ar.wg.Add(1)
	go func() {
		defer ar.wg.Done()

		interval := ar.GetAuctionInterval()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logger.Info("Auction auto-closer started with interval: " + interval.String())

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

// closeExpiredAuctions fecha todos os leilões que expiraram
func (ar *AuctionRepository) closeExpiredAuctions() {
	auctionDuration := ar.GetAuctionDuration()
	now := time.Now()
	expirationTime := now.Add(-auctionDuration)

	filter := bson.M{
		"status": auction_entity.Active,
		"timestamp": bson.M{
			"$lt": expirationTime.Unix(),
		},
	}

	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	result, err := ar.Collection.UpdateMany(ar.ctx, filter, update)
	if err != nil {
		logger.Error("Error closing expired auctions", err)
		return
	}

	if result.ModifiedCount > 0 {
		logger.Info("Closed " + strconv.FormatInt(result.ModifiedCount, 10) + " expired auctions")
	}
}

// Close encerra a goroutine de fechamento automático
func (ar *AuctionRepository) Close() {
	ar.cancel()
	ar.wg.Wait()
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
