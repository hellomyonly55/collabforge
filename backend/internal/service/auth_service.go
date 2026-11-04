package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"net/mail"
	"strconv"
	"strings"
	"taskflow/internal/dto"
	"taskflow/internal/infrastructure/password"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"time"
	"unicode/utf8"
)

type AuthService struct {
	Users     *repository.UserRepository
	Sessions  *repository.SessionRepository
	Log       *slog.Logger
	dummyHash string
}

func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository, log *slog.Logger) (*AuthService, error) {
	dummy, err := password.Hash("dummy-password-for-timing")
	if err != nil {
		return nil, err
	}
	return &AuthService{users, sessions, log, dummy}, nil
}
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", Fail(400, "VALIDATION_ERROR", "请输入有效邮箱")
	}
	for _, ch := range email {
		if ch > 127 {
			return "", Fail(400, "VALIDATION_ERROR", "邮箱请使用 ASCII 字符")
		}
	}
	return email, nil
}
func nickname(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 40 {
		return "", Fail(400, "VALIDATION_ERROR", "昵称长度应为 1 至 40 个字符")
	}
	return value, nil
}
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (dto.User, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return dto.User{}, err
	}
	name, err := nickname(req.Nickname)
	if err != nil {
		return dto.User{}, err
	}
	if err = password.Validate(req.Password); err != nil {
		return dto.User{}, Fail(400, "VALIDATION_ERROR", err.Error())
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return dto.User{}, Fail(500, "INTERNAL_ERROR", "密码处理失败")
	}
	u := &model.User{Email: email, Nickname: name, PasswordHash: hash, PlatformRole: "user", Status: "active", SessionVersion: 1}
	if err = s.Users.Create(ctx, u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.User{}, Fail(409, "EMAIL_EXISTS", "该邮箱已注册")
		}
		return dto.User{}, unavailable()
	}
	return dto.UserFrom(u), nil
}
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, oldToken string) (dto.User, string, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return dto.User{}, "", err
	}
	if err = password.Validate(req.Password); err != nil {
		return dto.User{}, "", Fail(400, "VALIDATION_ERROR", err.Error())
	}
	allowed, err := s.Sessions.Allow(ctx, "login:email:"+email, 10, 15*time.Minute)
	if err != nil {
		return dto.User{}, "", unavailable()
	}
	if !allowed {
		return dto.User{}, "", Fail(429, "RATE_LIMITED", "登录尝试过多，请 15 分钟后重试")
	}
	u, err := s.Users.ByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.User{}, "", unavailable()
	}
	hash := s.dummyHash
	if err == nil {
		hash = u.PasswordHash
	}
	match := password.Matches(hash, req.Password)
	if err != nil || !match || u.Status != "active" {
		return dto.User{}, "", Fail(401, "INVALID_CREDENTIALS", "邮箱或密码错误，或账号不可用")
	}
	if oldToken != "" {
		if err = s.Sessions.Delete(ctx, oldToken); err != nil {
			return dto.User{}, "", unavailable()
		}
	}
	token, err := s.Sessions.Create(ctx, u.ID, u.SessionVersion)
	if err != nil {
		return dto.User{}, "", unavailable()
	}
	return dto.UserFrom(u), token, nil
}
func (s *AuthService) Authenticate(ctx context.Context, token string) (*model.User, error) {
	session, err := s.Sessions.Lookup(ctx, token)
	if errors.Is(err, repository.ErrSessionNotFound) {
		return nil, unauthorized()
	}
	if err != nil {
		return nil, unavailable()
	}
	id, err := strconv.ParseUint(session.UserID, 10, 64)
	if err != nil || id == 0 {
		return nil, unauthorized()
	}
	u, err := s.Users.ByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, unauthorized()
	}
	if err != nil {
		return nil, unavailable()
	}
	if u.Status != "active" || u.SessionVersion != session.Version {
		return nil, unauthorized()
	}
	return u, nil
}
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := s.Sessions.Delete(ctx, token); err != nil {
		return unavailable()
	}
	return nil
}
func (s *AuthService) UpdateProfile(ctx context.Context, u *model.User, req dto.ProfileRequest) (dto.User, error) {
	name, err := nickname(req.Nickname)
	if err != nil {
		return dto.User{}, err
	}
	ok, err := s.Users.UpdateNickname(ctx, u, name)
	if err != nil {
		return dto.User{}, unavailable()
	}
	if !ok {
		return dto.User{}, unauthorized()
	}
	u.Nickname = name
	return dto.UserFrom(u), nil
}
func (s *AuthService) ChangePassword(ctx context.Context, u *model.User, req dto.PasswordRequest) error {
	if !password.Matches(u.PasswordHash, req.OldPassword) {
		return Fail(400, "OLD_PASSWORD_INCORRECT", "原密码不正确")
	}
	if err := password.Validate(req.NewPassword); err != nil {
		return Fail(400, "VALIDATION_ERROR", err.Error())
	}
	if req.NewPassword == req.OldPassword {
		return Fail(400, "VALIDATION_ERROR", "新密码不能与原密码相同")
	}
	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return Fail(500, "INTERNAL_ERROR", "密码处理失败")
	}
	ok, err := s.Users.UpdatePassword(ctx, u, hash)
	if err != nil {
		return unavailable()
	}
	if !ok {
		return Fail(409, "SESSION_CHANGED", "账号状态已变化，请重新登录")
	}
	// SQL session_version is the revocation authority. Redis cleanup may fail without reviving old sessions.
	cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = s.Sessions.RevokeAll(cleanup, u.ID); err != nil {
		s.Log.Warn("password changed; expired session cleanup deferred", "user_id", u.ID)
	}
	return nil
}
