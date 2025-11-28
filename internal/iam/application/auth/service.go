package auth

import (
	"context"
	"fmt"
	"tenant-crud-simply/internal/iam/application/auth/cache"
	"tenant-crud-simply/internal/iam/domain/user"
	"tenant-crud-simply/internal/infra/jwt"
	"tenant-crud-simply/internal/pkg/mailer"
	"tenant-crud-simply/internal/pkg/util"

	"github.com/google/uuid"
)

type implService struct {
	Repository Repository
}

type Service interface {
	Login(ctx context.Context, email, pwd string) (Login, error)
	RevokeAcessToken(ctx context.Context, token string) error
	GetAcessToken(ctx context.Context, token string) (AcessToken, error)
	CreateOTPCode(ctx context.Context, email string) error
	ValidateOTPCode(ctx context.Context, email, codeDst string) bool
	ChangeUserPwd(ctx context.Context, otpCode, email, pwd string) (bool, error)
}

func NewService(Repository Repository) Service {
	return &implService{
		Repository: Repository,
	}
}
func (s *implService) Login(ctx context.Context, email, pwd string) (Login, error) {
	rUser, err := user.MustUse().Service.Read(ctx, user.User{
		Email: email,
	})
	if err != nil {
		return Login{}, ErrPwdWrong
	}
	if err := util.UsePassword().Compare(rUser.Password, pwd); err != nil {
		return Login{}, ErrPwdWrong
	}
	var tenantID uuid.UUID
	if rUser.TenantUUID != nil {
		tenantID = *rUser.TenantUUID
	}
	token, expTime, err := jwt.Use().GenerateAccessToken(rUser.UUID, tenantID)
	if err != nil {
		return Login{}, err
	}
	AcessToken := AcessToken{UserUUID: &rUser.UUID, Token: token, Expiry: expTime}
	response := Login{
		User:       rUser,
		AcessToken: AcessToken,
	}

	// Invalida tokens anteriores para garantir sessão única (opcional, mas solicitado/implícito)
	_ = s.Repository.RevokeAllUserTokens(ctx, rUser.UUID.String())

	err = s.Repository.CreateAcessToken(ctx, AcessToken)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (s *implService) RevokeAcessToken(ctx context.Context, token string) error {
	return s.Repository.RevokeAcessToken(ctx, token)
}

func (s *implService) GetAcessToken(ctx context.Context, token string) (AcessToken, error) {
	return s.Repository.GetAcessToken(ctx, token)
}

func (s *implService) CreateOTPCode(ctx context.Context, email string) error {
	_, err := user.MustUse().Service.Read(ctx, user.User{Email: email})
	if err != nil {
		return err
	}
	_, found := cache.GetOTP(email)
	if found {
		return OTPCodeExist
	}
	otpCode, err := GenerateOTP(6)
	if err != nil {
		return err
	}
	mailService := mailer.Use()
	if mailService == nil {
		return mailer.ErrMailerNotInitialized
	}
	cache.SaveOTP(email, otpCode)
	err = mailService.SendRaw(
		email,
		"OTP Code",
		fmt.Sprintf("<h1>Seu código OTP é: %s</h1>", otpCode),
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *implService) ValidateOTPCode(ctx context.Context, email, codeDst string) bool {
	otpExist, found := cache.GetOTP(email)
	if !found {
		return false
	}
	if otpExist != codeDst {
		return false
	}

	return true
}

func (s *implService) ChangeUserPwd(ctx context.Context, otpCode, email, pwd string) (bool, error) {
	if !s.ValidateOTPCode(ctx, email, otpCode) {
		return false, OTPCodeWrong
	}
	cache.DeleteOTP(email)
	updUser := user.User{Email: email}
	userDst, err := user.MustUse().Service.Read(ctx, updUser)
	if err != nil {
		return false, err
	}
	updUser = user.User{UUID: userDst.UUID, Password: pwd}
	_, err = user.MustUse().Service.Update(ctx, updUser)
	if err != nil {
		return false, err
	}
	return true, nil
}
