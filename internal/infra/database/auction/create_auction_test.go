package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/entity/auction_entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestGetAuctionInterval(t *testing.T) {
	t.Run("returns default 5m when env not set", func(t *testing.T) {
		os.Unsetenv("AUCTION_INTERVAL")
		got := getAuctionInterval()
		if got != 5*time.Minute {
			t.Errorf("expected 5m, got %v", got)
		}
	})

	t.Run("parses duration from env", func(t *testing.T) {
		os.Setenv("AUCTION_INTERVAL", "30s")
		defer os.Unsetenv("AUCTION_INTERVAL")
		got := getAuctionInterval()
		if got != 30*time.Second {
			t.Errorf("expected 30s, got %v", got)
		}
	})

	t.Run("returns default on invalid value", func(t *testing.T) {
		os.Setenv("AUCTION_INTERVAL", "invalid")
		defer os.Unsetenv("AUCTION_INTERVAL")
		got := getAuctionInterval()
		if got != 5*time.Minute {
			t.Errorf("expected 5m, got %v", got)
		}
	})
}

func TestGetAuctionCloseCheckInterval(t *testing.T) {
	t.Run("returns default 10s when env not set", func(t *testing.T) {
		os.Unsetenv("AUCTION_CLOSE_CHECK_INTERVAL")
		got := getAuctionCloseCheckInterval()
		if got != 10*time.Second {
			t.Errorf("expected 10s, got %v", got)
		}
	})

	t.Run("parses duration from env", func(t *testing.T) {
		os.Setenv("AUCTION_CLOSE_CHECK_INTERVAL", "5s")
		defer os.Unsetenv("AUCTION_CLOSE_CHECK_INTERVAL")
		got := getAuctionCloseCheckInterval()
		if got != 5*time.Second {
			t.Errorf("expected 5s, got %v", got)
		}
	})
}

func TestCloseExpiredAuctions(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("executes UpdateMany to close expired auctions", func(mt *mtest.T) {
		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionInterval: 5 * time.Minute,
		}

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 3},
			{Key: "n", Value: 3},
		})

		repo.closeExpiredAuctions(context.Background())
	})

	mt.Run("handles UpdateMany error gracefully", func(mt *mtest.T) {
		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionInterval: 5 * time.Minute,
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    11000,
			Message: "mock update error",
		}))

		repo.closeExpiredAuctions(context.Background())
	})
}

func TestMonitorAuctionExpirationStopsOnContextCancel(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("goroutine exits when context is cancelled", func(mt *mtest.T) {
		for i := 0; i < 20; i++ {
			mt.AddMockResponses(bson.D{
				{Key: "ok", Value: 1},
				{Key: "nModified", Value: 0},
				{Key: "n", Value: 0},
			})
		}

		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionInterval: 1 * time.Second,
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})

		go func() {
			repo.monitorAuctionExpiration(ctx, 200*time.Millisecond)
			close(done)
		}()

		time.Sleep(800 * time.Millisecond)

		cancel()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("monitorAuctionExpiration goroutine did not exit after context cancellation")
		}
	})
}

func TestCreateAuctionWithMock(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("successfully inserts auction into collection", func(mt *mtest.T) {
		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionInterval: 5 * time.Minute,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		auctionEntity := &auction_entity.Auction{
			Id:          "test-auction-id",
			ProductName: "Test Product",
			Category:    "electronics",
			Description: "A test product description",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now(),
		}

		err := repo.CreateAuction(context.Background(), auctionEntity)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Message)
		}
	})

	mt.Run("returns error when InsertOne fails", func(mt *mtest.T) {
		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionInterval: 5 * time.Minute,
		}

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    11000,
			Message: "duplicate key",
		}))

		auctionEntity := &auction_entity.Auction{
			Id:          "duplicate-id",
			ProductName: "Duplicate Product",
			Category:    "electronics",
			Description: "This should fail with duplicate key",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now(),
		}

		err := repo.CreateAuction(context.Background(), auctionEntity)
		if err == nil {
			t.Fatal("expected error for duplicate insert, got nil")
		}
	})
}

func TestAutoCloseAuctionViaGoroutine(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("goroutine periodically invokes close on expired auctions", func(mt *mtest.T) {
		for i := 0; i < 30; i++ {
			mt.AddMockResponses(bson.D{
				{Key: "ok", Value: 1},
				{Key: "nModified", Value: 1},
				{Key: "n", Value: 1},
			})
		}

		os.Setenv("AUCTION_INTERVAL", "1s")
		os.Setenv("AUCTION_CLOSE_CHECK_INTERVAL", "200ms")
		defer os.Unsetenv("AUCTION_INTERVAL")
		defer os.Unsetenv("AUCTION_CLOSE_CHECK_INTERVAL")

		ctx, cancel := context.WithCancel(context.Background())

		repo := NewAuctionRepository(ctx, mt.DB)
		repo.Collection = mt.Coll

		time.Sleep(1 * time.Second)

		cancel()
		time.Sleep(200 * time.Millisecond)

	})
}
