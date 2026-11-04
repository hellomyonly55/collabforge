package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	redisclient "github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

const SessionTTL = 7 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	UserID  string `json:"user_id"`
	Version uint64 `json:"version"`
}
type SessionRepository struct {
	Client *redisclient.Client
	Prefix string
	TTL    time.Duration
}

func NewSessionRepository(client *redisclient.Client, prefix string) *SessionRepository {
	return &SessionRepository{Client: client, Prefix: prefix, TTL: SessionTTL}
}
func (r *SessionRepository) tokenKey(token string) (string, error) {
	if len(token) != 64 {
		return "", ErrSessionNotFound
	}
	if _, err := hex.DecodeString(token); err != nil {
		return "", ErrSessionNotFound
	}
	digest := sha256.Sum256([]byte(token))
	return r.Prefix + "session:" + hex.EncodeToString(digest[:]), nil
}
func (r *SessionRepository) userKey(id string) string { return r.Prefix + "user_sessions:" + id }
func (r *SessionRepository) Create(ctx context.Context, id, version uint64) (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes[:])
	key, _ := r.tokenKey(token)
	userID := strconv.FormatUint(id, 10)
	value, err := json.Marshal(Session{userID, version})
	if err != nil {
		return "", err
	}
	// A sorted index prunes expired sessions on each login, bounding index growth.
	_, err = r.Client.Eval(ctx, `local now=tonumber(ARGV[3]); redis.call('ZREMRANGEBYSCORE',KEYS[2],'-inf',now); redis.call('SET',KEYS[1],ARGV[1],'EX',ARGV[2]); redis.call('ZADD',KEYS[2],now+tonumber(ARGV[2]),KEYS[1]); redis.call('EXPIRE',KEYS[2],ARGV[2]); return 1`, []string{key, r.userKey(userID)}, string(value), int(r.TTL.Seconds()), time.Now().Unix()).Result()
	if err != nil {
		return "", err
	}
	return token, nil
}
func (r *SessionRepository) Lookup(ctx context.Context, token string) (Session, error) {
	key, err := r.tokenKey(token)
	if err != nil {
		return Session{}, err
	}
	raw, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redisclient.Nil) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err = json.Unmarshal([]byte(raw), &s); err != nil {
		return Session{}, err
	}
	return s, nil
}
func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	s, err := r.Lookup(ctx, token)
	if errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	key, _ := r.tokenKey(token)
	_, err = r.Client.Eval(ctx, `redis.call('DEL',KEYS[1]); redis.call('ZREM',KEYS[2],KEYS[1]); return 1`, []string{key, r.userKey(s.UserID)}).Result()
	return err
}
func (r *SessionRepository) RevokeAll(ctx context.Context, id uint64) error {
	_, err := r.Client.Eval(ctx, `local keys=redis.call('ZRANGE',KEYS[1],0,-1); for _,key in ipairs(keys) do redis.call('DEL',key) end; redis.call('DEL',KEYS[1]); return 1`, []string{r.userKey(strconv.FormatUint(id, 10))}).Result()
	return err
}

// Allow uses one Lua operation so a process crash cannot leave a counter without expiry.
func (r *SessionRepository) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	sum := sha256.Sum256([]byte(key))
	count, err := r.Client.Eval(ctx, `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],ARGV[1]) end; return n`, []string{r.Prefix + "limit:" + hex.EncodeToString(sum[:])}, int(window.Seconds())).Int64()
	return count <= int64(limit), err
}
