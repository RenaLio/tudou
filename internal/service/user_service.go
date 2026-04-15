package service

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	Create(ctx context.Context, req v1.CreateUserRequest) (*v1.UserResponse, error)
	BatchCreate(ctx context.Context, reqs []v1.CreateUserRequest) ([]v1.UserResponse, error)
	GetByID(ctx context.Context, id int64, withRelations bool) (*v1.UserWithRelationsResponse, error)
	GetByUsername(ctx context.Context, username string) (*v1.UserResponse, error)
	GetByEmail(ctx context.Context, email string) (*v1.UserResponse, error)
	List(ctx context.Context, req v1.ListUsersRequest) (*v1.ListResponse[v1.UserWithRelationsResponse], error)
	Update(ctx context.Context, id int64, req v1.UpdateUserRequest) (*v1.UserResponse, error)
	RecordLogin(ctx context.Context, id int64, ip string) error
	UpdateStatus(ctx context.Context, id int64, req v1.SetUserStatusRequest) (*v1.UserResponse, error)
	UpdatePassword(ctx context.Context, id int64, req v1.UpdateUserPasswordRequest) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	GenerateAccessToken(ctx context.Context, userID int64, ttl time.Duration) (string, error)
	ParseUserIDFromToken(ctx context.Context, token string) (int64, error)
	Login(ctx context.Context, req v1.UserLoginRequest, clientIP string) (*v1.UserLoginResponse, error)
}

type userService struct {
	*Service
	repo repository.UserRepo
}

func NewUserService(base *Service, repo repository.UserRepo) UserService {
	return &userService{
		Service: base,
		repo:    repo,
	}
}

func (s *userService) Create(ctx context.Context, req v1.CreateUserRequest) (*v1.UserResponse, error) {
	user, err := s.buildUserByCreateReq(req)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *userService) BatchCreate(ctx context.Context, reqs []v1.CreateUserRequest) ([]v1.UserResponse, error) {
	if len(reqs) == 0 {
		return []v1.UserResponse{}, nil
	}
	users := make([]*models.User, 0, len(reqs))
	for _, req := range reqs {
		user, err := s.buildUserByCreateReq(req)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := s.repo.BatchCreate(ctx, users); err != nil {
		return nil, err
	}
	resp := make([]v1.UserResponse, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		resp = append(resp, toUserResponse(user))
	}
	return resp, nil
}

func (s *userService) GetByID(ctx context.Context, id int64, withRelations bool) (*v1.UserWithRelationsResponse, error) {
	var (
		user *models.User
		err  error
	)
	if withRelations {
		user, err = s.repo.GetByIDWithRelations(ctx, id)
	} else {
		user, err = s.repo.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	resp := toUserWithRelationsResponse(user)
	return &resp, nil
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*v1.UserResponse, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*v1.UserResponse, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *userService) List(ctx context.Context, req v1.ListUsersRequest) (*v1.ListResponse[v1.UserWithRelationsResponse], error) {
	opt := repository.UserListOption{
		Page:          req.Page,
		PageSize:      req.PageSize,
		OrderBy:       req.OrderBy,
		Keyword:       req.Keyword,
		PreloadTokens: req.PreloadTokens,
		PreloadStats:  req.PreloadStats,
	}
	if req.Status != "" {
		status := models.UserStatus(req.Status)
		opt.Status = &status
	}
	if req.Role != "" {
		role := models.UserRole(req.Role)
		opt.Role = &role
	}

	items, total, err := s.repo.List(ctx, opt)
	if err != nil {
		return nil, err
	}
	respItems := make([]v1.UserWithRelationsResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, toUserWithRelationsResponse(item))
	}
	page, pageSize, _ := normalizePagination(req.Page, req.PageSize)
	return &v1.ListResponse[v1.UserWithRelationsResponse]{
		Total:    total,
		Items:    respItems,
		Page:     int64(page),
		PageSize: int64(pageSize),
	}, nil
}

func (s *userService) Update(ctx context.Context, id int64, req v1.UpdateUserRequest) (*v1.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	patchUserByUpdateReq(user, req)
	if strings.TrimSpace(user.Username) == "" {
		return nil, errors.New("username is required")
	}
	if err = s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *userService) RecordLogin(ctx context.Context, id int64, ip string) error {
	return s.repo.RecordLogin(ctx, id, ip)
}

func (s *userService) UpdateStatus(ctx context.Context, id int64, req v1.SetUserStatusRequest) (*v1.UserResponse, error) {
	if err := s.repo.UpdateStatus(ctx, id, req.Status); err != nil {
		return nil, err
	}
	return s.GetByIDAsUserResponse(ctx, id)
}

func (s *userService) UpdatePassword(ctx context.Context, id int64, req v1.UpdateUserPasswordRequest) error {
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return errors.New("password is required")
	}
	passwordHash, err := hashUserPassword(password)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, passwordHash)
}

func (s *userService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *userService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *userService) GenerateAccessToken(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	_ = ctx
	if userID <= 0 {
		return "", errors.New("invalid user id")
	}
	j := s.JWT()
	if j == nil {
		return "", errors.New("jwt service is nil")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return j.GenToken(int64ToString(userID), time.Now().Add(ttl))
}

func (s *userService) ParseUserIDFromToken(ctx context.Context, token string) (int64, error) {
	_ = ctx
	j := s.JWT()
	if j == nil {
		return 0, errors.New("jwt service is nil")
	}
	claims, err := j.ParseToken(token)
	if err != nil {
		return 0, err
	}
	if claims.UserId == "" {
		return 0, errors.New("user id is empty in token claims")
	}
	return stringToInt64(claims.UserId)
}

func (s *userService) Login(ctx context.Context, req v1.UserLoginRequest, clientIP string) (*v1.UserLoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return nil, errors.New("username/password are required")
	}
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}
	if !verifyUserPassword(user.Password, password) {
		return nil, errors.New("invalid username or password")
	}

	const ttl = 24 * time.Hour
	token, err := s.GenerateAccessToken(ctx, user.ID, ttl)
	if err != nil {
		return nil, err
	}
	if err = s.repo.RecordLogin(ctx, user.ID, clientIP); err != nil {
		return nil, err
	}

	return &v1.UserLoginResponse{
		AccessToken: token,
		ExpiresIn:   int64(ttl.Seconds()),
		User:        toUserResponse(user),
	}, nil
}

func (s *userService) GetByIDAsUserResponse(ctx context.Context, id int64) (*v1.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *userService) buildUserByCreateReq(req v1.CreateUserRequest) (*models.User, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return nil, errors.New("username/password are required")
	}
	passwordHash, err := hashUserPassword(password)
	if err != nil {
		return nil, err
	}
	id := s.NextID()
	if id <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	user := &models.User{
		ID:       id,
		Username: username,
		Password: passwordHash,
		Email:    strings.TrimSpace(req.Email),
		Phone:    strings.TrimSpace(req.Phone),
		Nickname: strings.TrimSpace(req.Nickname),
		Avatar:   strings.TrimSpace(req.Avatar),
		Settings: req.Settings,
	}
	if req.Status == nil {
		user.Status = models.UserStatusEnabled
	} else {
		user.Status = *req.Status
	}
	if req.Role == nil {
		user.Role = models.UserRoleUser
	} else {
		user.Role = *req.Role
	}
	return user, nil
}

func hashUserPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifyUserPassword(storedPassword, plainPassword string) bool {
	storedPassword = strings.TrimSpace(storedPassword)
	if storedPassword == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(plainPassword)) == nil
}

func patchUserByUpdateReq(user *models.User, req v1.UpdateUserRequest) {
	if user == nil {
		return
	}
	if req.Email != nil {
		user.Email = strings.TrimSpace(*req.Email)
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Nickname != nil {
		user.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.Avatar != nil {
		user.Avatar = strings.TrimSpace(*req.Avatar)
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Settings != nil {
		user.Settings = *req.Settings
	}
	if req.LastLoginIP != nil {
		user.LastLoginIP = strings.TrimSpace(*req.LastLoginIP)
	}
}

func toUserResponse(user *models.User) v1.UserResponse {
	if user == nil {
		return v1.UserResponse{}
	}
	return v1.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Phone:       user.Phone,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Status:      user.Status,
		Role:        user.Role,
		LastLoginAt: user.LastLoginAt,
		LastLoginIP: user.LastLoginIP,
		LoginCount:  user.LoginCount,
		Settings:    user.Settings,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func toUserWithRelationsResponse(user *models.User) v1.UserWithRelationsResponse {
	if user == nil {
		return v1.UserWithRelationsResponse{}
	}
	resp := v1.UserWithRelationsResponse{
		UserResponse: toUserResponse(user),
	}
	if user.Stats.UserID != 0 {
		stats := user.Stats
		resp.Stats = &stats
	}
	if len(user.Tokens) > 0 {
		resp.Tokens = make([]v1.TokenResponse, 0, len(user.Tokens))
		for i := range user.Tokens {
			resp.Tokens = append(resp.Tokens, toTokenResponse(&user.Tokens[i]))
		}
	}
	return resp
}
