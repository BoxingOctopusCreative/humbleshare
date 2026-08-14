package service

import (
	"context"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
)

func TestCodeServiceReportDead(t *testing.T) {
	t.Run("empty user", func(t *testing.T) {
		svc := NewCodeService(&stubCodeRepo{})
		err := svc.ReportDead(context.Background(), uuid.New(), "  ")
		if err != repository.ErrNotClaimant {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("passes trimmed user id", func(t *testing.T) {
		id := uuid.New()
		var gotID uuid.UUID
		var gotUser string
		svc := NewCodeService(&stubCodeRepo{
			reportDeadFn: func(_ context.Context, codeID uuid.UUID, claimedByUserID string) error {
				gotID = codeID
				gotUser = claimedByUserID
				return nil
			},
		})
		if err := svc.ReportDead(context.Background(), id, "  user-1  "); err != nil {
			t.Fatal(err)
		}
		if gotID != id {
			t.Fatalf("id=%s want=%s", gotID, id)
		}
		if gotUser != "user-1" {
			t.Fatalf("user=%q", gotUser)
		}
	})
}
