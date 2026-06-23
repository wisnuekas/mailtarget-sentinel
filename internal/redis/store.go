package redisstore

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wisnuekas/mailtarget-sentinel/internal/alert"
	"github.com/redis/go-redis/v9"
)

const (
	settingsKey             = "sentinel:settings"
	alertLockPrefix         = "sentinel:alert:lock:"
	sendingIPAlertLockPrefix = "sentinel:alert:sending-ip:"
	killTokenPrefix         = "sentinel:kill:"
	resumeTokenPrefix       = "sentinel:resume:"
)

type Settings struct {
	MinVolume              uint64  `json:"min_volume"`
	BounceRateThresholdPct float64 `json:"bounce_rate_threshold_pct"`
	SpamRateThresholdPct   float64 `json:"spam_rate_threshold_pct"`
	AlertCooldownMinutes   int     `json:"alert_cooldown_minutes"`
	CompanyID              *int32  `json:"company_id"`
}

func DefaultSettings() Settings {
	return Settings{
		MinVolume:              100,
		BounceRateThresholdPct: 5.0,
		SpamRateThresholdPct:   1.0,
		AlertCooldownMinutes:   30,
	}
}

type Store struct {
	client     *redis.Client
	hmacSecret []byte
	tokenTTL   time.Duration
}

func NewStore(addr, password string, db int, hmacSecret string, tokenTTL time.Duration) *Store {
	return &Store{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		hmacSecret: []byte(hmacSecret),
		tokenTTL:   tokenTTL,
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) GetSettings(ctx context.Context) (Settings, error) {
	data, err := s.client.Get(ctx, settingsKey).Bytes()
	if err == redis.Nil {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("get settings: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("unmarshal settings: %w", err)
	}
	return settings, nil
}

func (s *Store) UpdateSettings(ctx context.Context, partial Settings) (Settings, error) {
	current, err := s.GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}

	if partial.MinVolume > 0 {
		current.MinVolume = partial.MinVolume
	}
	if partial.BounceRateThresholdPct > 0 {
		current.BounceRateThresholdPct = partial.BounceRateThresholdPct
	}
	if partial.SpamRateThresholdPct > 0 {
		current.SpamRateThresholdPct = partial.SpamRateThresholdPct
	}
	if partial.AlertCooldownMinutes > 0 {
		current.AlertCooldownMinutes = partial.AlertCooldownMinutes
	}
	if partial.CompanyID != nil {
		current.CompanyID = partial.CompanyID
	}

	data, err := json.Marshal(current)
	if err != nil {
		return Settings{}, fmt.Errorf("marshal settings: %w", err)
	}

	if err := s.client.Set(ctx, settingsKey, data, 0).Err(); err != nil {
		return Settings{}, fmt.Errorf("set settings: %w", err)
	}

	return current, nil
}

func (s *Store) TryAcquireAlertLock(ctx context.Context, subAccountID int32, cooldown time.Duration) (bool, error) {
	key := fmt.Sprintf("%s%d", alertLockPrefix, subAccountID)
	ok, err := s.client.SetNX(ctx, key, "1", cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("acquire alert lock: %w", err)
	}
	return ok, nil
}

func (s *Store) TryAcquireSendingIPAlertLock(ctx context.Context, sendingIP string, cooldown time.Duration) (bool, error) {
	key := fmt.Sprintf("%s%s", sendingIPAlertLockPrefix, sendingIP)
	ok, err := s.client.SetNX(ctx, key, "1", cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("acquire sending IP alert lock: %w", err)
	}
	return ok, nil
}

func (s *Store) ExtendAlertLock(ctx context.Context, subAccountID int32, cooldown time.Duration) error {
	key := fmt.Sprintf("%s%d", alertLockPrefix, subAccountID)
	return s.client.Expire(ctx, key, cooldown).Err()
}

type KillTokenPayload struct {
	SubAccountID int32            `json:"sub_account_id"`
	Alert        alert.AnomalyAlert `json:"alert"`
}

func (s *Store) CreateKillToken(ctx context.Context, payload KillTokenPayload) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	raw := fmt.Sprintf("%d:%s:%d", payload.SubAccountID, hex.EncodeToString(nonce), time.Now().Unix())
	mac := hmac.New(sha256.New, s.hmacSecret)
	mac.Write([]byte(raw))
	token := hex.EncodeToString(mac.Sum(nil))[:32]

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal kill token: %w", err)
	}

	key := killTokenPrefix + token
	if err := s.client.Set(ctx, key, data, s.tokenTTL).Err(); err != nil {
		return "", fmt.Errorf("store kill token: %w", err)
	}

	return token, nil
}

func (s *Store) ConsumeKillToken(ctx context.Context, token string) (*KillTokenPayload, error) {
	key := killTokenPrefix + token
	data, err := s.client.GetDel(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("invalid or expired token")
	}
	if err != nil {
		return nil, fmt.Errorf("get kill token: %w", err)
	}

	var payload KillTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal kill token: %w", err)
	}

	return &payload, nil
}

type ResumeTokenPayload struct {
	SubAccountID int32  `json:"sub_account_id"`
	CompanyID    int32  `json:"company_id"`
	AlertID      string `json:"alert_id,omitempty"`
}

func (s *Store) CreateResumeToken(ctx context.Context, payload ResumeTokenPayload) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	raw := fmt.Sprintf("resume:%d:%s:%d", payload.SubAccountID, hex.EncodeToString(nonce), time.Now().Unix())
	mac := hmac.New(sha256.New, s.hmacSecret)
	mac.Write([]byte(raw))
	token := hex.EncodeToString(mac.Sum(nil))[:32]

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal resume token: %w", err)
	}

	key := resumeTokenPrefix + token
	if err := s.client.Set(ctx, key, data, s.tokenTTL).Err(); err != nil {
		return "", fmt.Errorf("store resume token: %w", err)
	}

	return token, nil
}

func (s *Store) ConsumeResumeToken(ctx context.Context, token string) (*ResumeTokenPayload, error) {
	key := resumeTokenPrefix + token
	data, err := s.client.GetDel(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("invalid or expired token")
	}
	if err != nil {
		return nil, fmt.Errorf("get resume token: %w", err)
	}

	var payload ResumeTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal resume token: %w", err)
	}

	return &payload, nil
}
