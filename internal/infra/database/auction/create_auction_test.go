package auction

import (
	"context"
	"github.com/Alef-Daniel/challenge-go-auctions/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	mongoURL := os.Getenv("MONGODB_TEST_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://admin:admin@localhost:27017/?authSource=admin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Skipf("mongodb not available: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("mongodb not available: %v", err)
	}

	database := client.Database("auctions_test_" + uuid.NewString()[:8])
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return database
}

func TestCreateAuction_ClosesAutomaticallyAfterDuration(t *testing.T) {
	t.Setenv("AUCTION_DURATION", "2s")

	database := setupTestDatabase(t)
	repository := NewAuctionRepository(database)
	ctx := context.Background()

	auction, internalErr := auction_entity.CreateAuction(
		"Notebook", "Eletronicos", "Notebook usado em bom estado", auction_entity.Used)
	if internalErr != nil {
		t.Fatalf("unexpected error creating auction entity: %v", internalErr)
	}

	if err := repository.CreateAuction(ctx, auction); err != nil {
		t.Fatalf("unexpected error creating auction: %v", err)
	}

	created, findErr := repository.FindAuctionById(ctx, auction.Id)
	if findErr != nil {
		t.Fatalf("unexpected error finding auction: %v", findErr)
	}
	if created.Status != auction_entity.Active {
		t.Fatalf("expected status Active right after creation, got %v", created.Status)
	}

	time.Sleep(3 * time.Second)

	closed, findErr := repository.FindAuctionById(ctx, auction.Id)
	if findErr != nil {
		t.Fatalf("unexpected error finding auction: %v", findErr)
	}
	if closed.Status != auction_entity.Completed {
		t.Fatalf("expected status Completed after duration, got %v", closed.Status)
	}
}

func TestGetAuctionDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		interval string
		expected time.Duration
	}{
		{"uses AUCTION_DURATION", "30s", "", 30 * time.Second},
		{"falls back to AUCTION_INTERVAL", "", "45s", 45 * time.Second},
		{"prefers AUCTION_DURATION", "10s", "45s", 10 * time.Second},
		{"defaults when invalid", "abc", "", defaultAuctionDuration},
		{"defaults when empty", "", "", defaultAuctionDuration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUCTION_DURATION", tt.duration)
			t.Setenv("AUCTION_INTERVAL", tt.interval)

			if got := GetAuctionDuration(); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
