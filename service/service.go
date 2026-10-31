// Package service - gRPC-слой: принимает запросы, проверяет данные
// и обращается к storage.
package service

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "game.go/gamepb" // замените YOUR_MODULE на имя модуля из go.mod
	"game.go/storage"
)

type Leaderboard struct {
	pb.UnimplementedLeaderboardServer
	store *storage.Storage
}

func New(store *storage.Storage) *Leaderboard {
	return &Leaderboard{store: store}
}

func (l *Leaderboard) SubmitScore(ctx context.Context, req *pb.SubmitScoreRequest) (*pb.SubmitScoreResponse, error) {
	name := strings.TrimSpace(req.Player)
	if name == "" || len([]rune(name)) > 20 || req.Score < 0 {
		return nil, status.Error(codes.InvalidArgument, "некорректное имя или счёт")
	}

	best, rank, err := l.store.SubmitScore(ctx, name, req.Score)
	if err != nil {
		log.Println("SubmitScore:", err)
		return nil, status.Error(codes.Internal, "ошибка базы данных")
	}

	log.Printf("результат: %s = %d (лучший %d, место %d)", name, req.Score, best, rank)
	return &pb.SubmitScoreResponse{Rank: rank, Best: best}, nil
}

func (l *Leaderboard) TopScores(ctx context.Context, req *pb.TopScoresRequest) (*pb.TopScoresResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	entries, err := l.store.Top(ctx, limit)
	if err != nil {
		log.Println("TopScores:", err)
		return nil, status.Error(codes.Internal, "ошибка базы данных")
	}

	out := make([]*pb.ScoreEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, &pb.ScoreEntry{Player: e.Player, Score: e.Score, UpdatedAt: e.UpdatedAt})
	}
	return &pb.TopScoresResponse{Entries: out}, nil
}
