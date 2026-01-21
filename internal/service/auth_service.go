package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
	"golang.org/x/crypto/bcrypt"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type AuthService struct {
	userRepo   repository.UserRepository
	k8sClient  kubernetes.Interface
	jwtSecret  []byte
	jwtExpires time.Duration
}

func NewAuthService(userRepo repository.UserRepository, k8sClient kubernetes.Interface, secret string) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		k8sClient:  k8sClient,
		jwtSecret:  []byte(secret),
		jwtExpires: 24 * time.Hour,
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password string) (*models.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Username,
		"role": string(user.Role),
		"exp":  time.Now().Add(s.jwtExpires).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken 验证令牌 (JWT, Static, or K8s)
func (s *AuthService) ValidateToken(ctx context.Context, tokenStr string) (*models.User, error) {
	// 1. 尝试作为 JWT 验证 (Web UI)
	user, err := s.validateJWT(tokenStr)
	if err == nil {
		return user, nil
	}

	// 2. 尝试作为静态 API Token 验证 (其他项目)
	user, err = s.validateStaticToken(ctx, tokenStr)
	if err == nil {
		return user, nil
	}

	// 3. 尝试作为 K8s Token 验证 (kubectl)
	user, err = s.validateK8sToken(ctx, tokenStr)
	if err == nil {
		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) validateJWT(tokenStr string) (*models.User, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid jwt")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	// 注意: 这里为了简单不每次查数据库，如果需要角色即时生效则需要查库
	userID := int64(claims["sub"].(float64))
	username := claims["name"].(string)
	role := models.UserRole(claims["role"].(string))

	return &models.User{
		ID:       userID,
		Username: username,
		Role:     role,
	}, nil
}

func (s *AuthService) validateStaticToken(ctx context.Context, tokenStr string) (*models.User, error) {
	token, err := s.userRepo.GetToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return s.userRepo.GetUser(ctx, token.UserID)
}

func (s *AuthService) validateK8sToken(ctx context.Context, tokenStr string) (*models.User, error) {
	if s.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not initialized")
	}

	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: tokenStr,
		},
	}

	result, err := s.k8sClient.AuthenticationV1().TokenReviews().Create(ctx, tr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	if !result.Status.Authenticated {
		return nil, fmt.Errorf("k8s authentication failed: %s", result.Status.Error)
	}

	// K8s 验证通过，映射为 Viewer 角色
	return &models.User{
		Username: result.Status.User.Username,
		Role:     models.RoleViewer, // 强制映射为 Viewer
	}, nil
}
