package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDatabase() (*mongo.Database, func()) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	db := client.Database("test_auction_db")

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestAuctionAutoCloser(t *testing.T) {
	// Configura variáveis de ambiente para o teste
	os.Setenv("AUCTION_DURATION", "2") // 2 segundos
	os.Setenv("AUCTION_INTERVAL", "1") // 1 segundo
	defer os.Unsetenv("AUCTION_DURATION")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDatabase()
	defer cleanup()

	auctionRepo := NewAuctionRepository(db)
	defer auctionRepo.Close()

	// Cria um leilão que deve expirar em 2 segundos
	auctionEntity := &auction_entity.Auction{
		Id:          uuid.New().String(),
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	internalErr := auctionRepo.CreateAuction(context.Background(), auctionEntity)
	assert.Nil(t, internalErr)

	// Verifica se o leilão foi criado como ativo
	var auction AuctionEntityMongo
	err := auctionRepo.Collection.FindOne(context.Background(), bson.M{"_id": auctionEntity.Id}).Decode(&auction)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, auction.Status)

	// Aguarda o leilão expirar e ser fechado automaticamente
	time.Sleep(4 * time.Second)

	// Verifica se o leilão foi fechado automaticamente
	err = auctionRepo.Collection.FindOne(context.Background(), bson.M{"_id": auctionEntity.Id}).Decode(&auction)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, auction.Status)
}

func TestGetAuctionDuration(t *testing.T) {
	db, cleanup := setupTestDatabase()
	defer cleanup()

	auctionRepo := NewAuctionRepository(db)
	defer auctionRepo.Close()

	// Testa com valor padrão
	duration := auctionRepo.GetAuctionDuration()
	assert.Equal(t, 300*time.Second, duration)

	// Testa com variável de ambiente
	os.Setenv("AUCTION_DURATION", "120")
	duration = auctionRepo.GetAuctionDuration()
	assert.Equal(t, 120*time.Second, duration)
	os.Unsetenv("AUCTION_DURATION")
}

func TestGetAuctionInterval(t *testing.T) {
	db, cleanup := setupTestDatabase()
	defer cleanup()

	auctionRepo := NewAuctionRepository(db)
	defer auctionRepo.Close()

	// Testa com valor padrão
	interval := auctionRepo.GetAuctionInterval()
	assert.Equal(t, 60*time.Second, interval)

	// Testa com variável de ambiente
	os.Setenv("AUCTION_INTERVAL", "30")
	interval = auctionRepo.GetAuctionInterval()
	assert.Equal(t, 30*time.Second, interval)
	os.Unsetenv("AUCTION_INTERVAL")
}

func TestCloseExpiredAuctions(t *testing.T) {
	// Configura variáveis de ambiente para o teste
	os.Setenv("AUCTION_DURATION", "1") // 1 segundo
	defer os.Unsetenv("AUCTION_DURATION")

	db, cleanup := setupTestDatabase()
	defer cleanup()

	auctionRepo := NewAuctionRepository(db)
	defer auctionRepo.Close()

	// Cria um leilão que já deve estar expirado
	pastTime := time.Now().Add(-2 * time.Second)
	auctionEntity := &auction_entity.Auction{
		Id:          uuid.New().String(),
		ProductName: "Expired Product",
		Category:    "Test Category",
		Description: "Test Description",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   pastTime,
	}

	internalErr := auctionRepo.CreateAuction(context.Background(), auctionEntity)
	assert.Nil(t, internalErr)

	// Executa o fechamento de leilões expirados
	auctionRepo.closeExpiredAuctions()

	// Verifica se o leilão foi fechado
	var auction AuctionEntityMongo
	err := auctionRepo.Collection.FindOne(context.Background(), bson.M{"_id": auctionEntity.Id}).Decode(&auction)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, auction.Status)
}
