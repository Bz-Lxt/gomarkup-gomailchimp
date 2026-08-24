package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lumen/relay/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	QSend  = "lumen:q:send"
	QDelay = "lumen:z:delay"
	QDLQ   = "lumen:q:dlq"
	QIdem  = "lumen:idem:"
)

type Queue struct {
	rdb *redis.Client
}

func NewQueue(rdb *redis.Client) *Queue { return &Queue{rdb: rdb} }

func (q *Queue) Push(ctx context.Context, job domain.SendJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.LPush(ctx, QSend, b).Err()
}

func (q *Queue) PushMany(ctx context.Context, jobs []domain.SendJob) error {
	if len(jobs) == 0 {
		return nil
	}
	vals := make([]any, 0, len(jobs))
	for _, j := range jobs {
		b, err := json.Marshal(j)
		if err != nil {
			return err
		}
		vals = append(vals, b)
	}
	return q.rdb.LPush(ctx, QSend, vals...).Err()
}

func (q *Queue) Delay(ctx context.Context, job domain.SendJob, at time.Time) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.ZAdd(ctx, QDelay, redis.Z{Score: float64(at.UnixMilli()), Member: b}).Err()
}

func (q *Queue) Pop(ctx context.Context, wait time.Duration) (domain.SendJob, bool, error) {
	res, err := q.rdb.BRPop(ctx, wait, QSend).Result()
	if err == redis.Nil {
		return domain.SendJob{}, false, nil
	}
	if err != nil {
		return domain.SendJob{}, false, err
	}
	if len(res) < 2 {
		return domain.SendJob{}, false, nil
	}
	var job domain.SendJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return domain.SendJob{}, false, err
	}
	return job, true, nil
}

func (q *Queue) PromoteDue(ctx context.Context, now time.Time) (int, error) {
	max := fmt.Sprintf("%d", now.UnixMilli())
	vals, err := q.rdb.ZRangeByScore(ctx, QDelay, &redis.ZRangeBy{Min: "-inf", Max: max, Count: 500}).Result()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, v := range vals {
		if err := q.rdb.ZRem(ctx, QDelay, v).Err(); err != nil {
			return n, err
		}
		if err := q.rdb.LPush(ctx, QSend, v).Err(); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (q *Queue) Dead(ctx context.Context, job domain.SendJob) error {
	b, _ := json.Marshal(job)
	return q.rdb.LPush(ctx, QDLQ, b).Err()
}

func (q *Queue) Depth(ctx context.Context) (send, delay, dlq int64, err error) {
	send, err = q.rdb.LLen(ctx, QSend).Result()
	if err != nil {
		return
	}
	delay, err = q.rdb.ZCard(ctx, QDelay).Result()
	if err != nil {
		return
	}
	dlq, err = q.rdb.LLen(ctx, QDLQ).Result()
	return
}

func (q *Queue) ClaimIdem(ctx context.Context, messageID string, ttl time.Duration) (bool, error) {
	ok, err := q.rdb.SetNX(ctx, QIdem+messageID, "1", ttl).Result()
	return ok, err
}
