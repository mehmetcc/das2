package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mehmetcc/das2/internal/config"
	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
)

type TokenService interface {
	Issue(ctx context.Context, p *person.Person, sessionID id.SessionID, meta IssueMeta) (*IssueResult, error)
	ValidateAccess(ctx context.Context, tokenString string) (*Claims, error)
	Refresh(ctx context.Context, presentedRefresh string, meta IssueMeta) (*IssueResult, error)
}

type IssueResult struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type IssueMeta struct {
	UserAgent string
	IP        string
	DeviceID  string
}

type tokenService struct {
	logger      *zap.Logger
	refreshRepo RefreshTokenRepo
	cfg         *config.JWTConfig
	signingAlg  jwt.SigningMethod
}

func NewTokenService(logger *zap.Logger, refreshRepo RefreshTokenRepo, cfg *config.JWTConfig) TokenService {
	return &tokenService{
		logger:      logger,
		refreshRepo: refreshRepo,
		cfg:         cfg,
		signingAlg:  jwt.SigningMethodHS256,
	}
}

func (s *tokenService) Issue(ctx context.Context, p *person.Person, sessionID id.SessionID, meta IssueMeta) (*IssueResult, error) {
	issuedAt := time.Now().UTC()
	accessExp := issuedAt.Add(s.cfg.AccessTTL)
	refreshExp := issuedAt.Add(s.cfg.RefreshTTL)
	claims := &Claims{
		Sub:  p.PublicID,
		SID:  sessionID,
		Role: p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{s.cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(accessExp),
			NotBefore: jwt.NewNumericDate(issuedAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ID:        s.generateJTI(),
		},
	}

	jwtToken := jwt.NewWithClaims(s.signingAlg, claims)
	jwtToken.Header["kid"] = s.cfg.JWTKID
	accessToken, err := jwtToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		s.logger.Error("failed to sign access token", zap.Error(err))
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, err
	}
	refreshHash := s.hashToken(refreshToken)

	_, err = s.refreshRepo.Create(ctx, RefreshTokenDTO{
		PersonID:  p.ID,
		SessionID: sessionID,
		TokenHash: refreshHash,
		ExpiresAt: refreshExp,
		UserAgent: meta.UserAgent,
		IP:        meta.IP,
		DeviceID:  meta.DeviceID,
	})
	if err != nil {
		return nil, err
	}

	return &IssueResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExp,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (s *tokenService) ValidateAccess(ctx context.Context, tokenString string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{s.signingAlg.Alg()}),
	)

	var claims Claims
	tkn, err := parser.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !tkn.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != s.cfg.JWTIssuer {
		return nil, errors.New("invalid issuer")
	}

	{
		ok := false
		for _, aud := range claims.Audience {
			if aud == s.cfg.JWTAudience {
				ok = true
				break
			}
		}
		if !ok {
			return nil, errors.New("invalid audience")
		}
	}
	return &claims, nil
}

// Refresh validates the presented refresh token, rotates it, and returns new tokens.
func (s *tokenService) Refresh(ctx context.Context, presentedRefresh string, meta IssueMeta) (*IssueResult, error) {
	if presentedRefresh == "" {
		return nil, errors.New("missing refresh token")
	}
	now := time.Now().UTC()
	hash := s.hashToken(presentedRefresh)
	rec, err := s.refreshRepo.FindActiveByHash(ctx, hash, now)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		// probable reuse of an already-rotated token: we cannot map hash to id without storing last hash.
		// Defensive strategy: do nothing specific here; caller can choose to log and return 401.
		return nil, errors.New("invalid refresh token")
	}

	// Build person for claims
	p := &person.Person{ID: rec.PersonID, PublicID: rec.PersonPublicID, Role: rec.PersonRole}

	// Issue new access + refresh, and mark old as rotated atomically via repo
	issuedAt := now
	accessExp := issuedAt.Add(s.cfg.AccessTTL)
	refreshExp := issuedAt.Add(s.cfg.RefreshTTL)

	claims := &Claims{
		Sub:  p.PublicID,
		SID:  rec.SessionID,
		Role: p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{s.cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(accessExp),
			NotBefore: jwt.NewNumericDate(issuedAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ID:        s.generateJTI(),
		},
	}

	jwtToken := jwt.NewWithClaims(s.signingAlg, claims)
	jwtToken.Header["kid"] = s.cfg.JWTKID
	accessToken, err := jwtToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	newRefresh, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}
	newHash := s.hashToken(newRefresh)

	_, err = s.refreshRepo.RotateCreateNext(ctx, rec.ID, RefreshTokenDTO{
		PersonID:  rec.PersonID,
		SessionID: rec.SessionID,
		TokenHash: newHash,
		ExpiresAt: refreshExp,
		UserAgent: meta.UserAgent,
		IP:        meta.IP,
		DeviceID:  meta.DeviceID,
	})
	if err != nil {
		return nil, err
	}

	return &IssueResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExp,
		RefreshToken:     newRefresh,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (s *tokenService) generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *tokenService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *tokenService) hashToken(str string) string {
	h := sha256.Sum256([]byte(str))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
