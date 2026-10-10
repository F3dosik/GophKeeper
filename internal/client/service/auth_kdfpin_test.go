package service_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/kdfpin"
	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Дешёвые допустимые параметры для тестов: strong отличается от weak числом проходов.
var (
	weakKDF   = domain.KDFParams{Time: 1, MemoryKiB: 19 * 1024, Threads: 1}
	strongKDF = domain.KDFParams{Time: 2, MemoryKiB: 19 * 1024, Threads: 1}
)

const testServer = "srv:50051"

func newPinnedService(t *testing.T, client *mocks.AuthClient) (service.AuthService, *kdfpin.Store) {
	t.Helper()
	dir := t.TempDir()
	pins := kdfpin.New(filepath.Join(dir, kdfpin.FileName))
	svc := service.NewAuthService(client, filepath.Join(dir, "session"), nil, service.WithKDFPins(pins, testServer))
	return svc, pins
}

func TestAuthService_KDFPins_FirstLoginRemembers(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("saltsaltsaltsalt"), strongKDF, nil)
	client.On("Login", mock.Anything, mock.Anything).Return("token", false, nil)
	svc, pins := newPinnedService(t, client)

	_, err := svc.Login(context.Background(), "alice", "password")
	require.NoError(t, err)

	got, ok, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, strongKDF, got)
}

// Подменённый сервер присылает более слабые параметры: клиент не выводит ключ и не
// отправляет его на сервер (Login не вызывается — мок упал бы на неожиданном вызове).
func TestAuthService_KDFPins_RejectsDowngrade(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("othersaltothersa"), weakKDF, nil)
	svc, pins := newPinnedService(t, client)
	require.NoError(t, pins.Remember(testServer, "alice", strongKDF))

	ctx := context.Background()
	_, err := svc.Login(ctx, "alice", "password")
	var downgrade *domain.KDFDowngradeError
	require.ErrorAs(t, err, &downgrade)
	assert.Equal(t, strongKDF, downgrade.Known)
	assert.Equal(t, weakKDF, downgrade.Received)

	_, err = svc.Unlock(ctx, "alice", "password")
	assert.ErrorIs(t, err, domain.ErrKDFDowngrade)
	err = svc.DeleteAccount(ctx, "alice", "password")
	assert.ErrorIs(t, err, domain.ErrKDFDowngrade)
	err = svc.ChangePassword(ctx, "alice", "password", "new-password", strongKDF, nil)
	assert.ErrorIs(t, err, domain.ErrKDFDowngrade)

	got, _, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	assert.Equal(t, strongKDF, got, "запомненные параметры не меняются")
}

// Более сильные параметры (смена пароля на другом устройстве) принимаются и запоминаются.
func TestAuthService_KDFPins_AcceptsStronger(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("saltsaltsaltsalt"), strongKDF, nil)
	client.On("Login", mock.Anything, mock.Anything).Return("token", false, nil)
	svc, pins := newPinnedService(t, client)
	require.NoError(t, pins.Remember(testServer, "alice", weakKDF))

	_, err := svc.Unlock(context.Background(), "alice", "password")
	require.NoError(t, err)

	got, _, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	assert.Equal(t, strongKDF, got)
}

// После подтверждения пользователя (ForgetKDFParams) ослабленные параметры принимаются.
func TestAuthService_KDFPins_ForgetAllowsWeaker(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("saltsaltsaltsalt"), weakKDF, nil)
	client.On("Login", mock.Anything, mock.Anything).Return("token", false, nil)
	svc, pins := newPinnedService(t, client)
	require.NoError(t, pins.Remember(testServer, "alice", strongKDF))

	require.NoError(t, svc.ForgetKDFParams("alice"))
	_, err := svc.Login(context.Background(), "alice", "password")
	require.NoError(t, err)

	got, _, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	assert.Equal(t, weakKDF, got)
}

// Параметры, выбранные при смене пароля на этом устройстве, запоминаются, даже если слабее.
func TestAuthService_KDFPins_ChangePasswordRemembersNew(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("saltsaltsaltsalt"), strongKDF, nil)
	client.On("ChangePassword", mock.Anything, mock.Anything, mock.Anything).Return("token", nil)
	svc, pins := newPinnedService(t, client)
	require.NoError(t, pins.Remember(testServer, "alice", strongKDF))

	err := svc.ChangePassword(context.Background(), "alice", "password", "new-password", weakKDF, nil)
	require.NoError(t, err)

	got, _, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	assert.Equal(t, weakKDF, got)
}

func TestAuthService_KDFPins_RegisterAndDelete(t *testing.T) {
	client := mocks.NewAuthClient(t)
	client.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, strongKDF, false).Return(nil, nil)
	client.On("GetSalt", mock.Anything, "alice").Return([]byte("saltsaltsaltsalt"), strongKDF, nil)
	client.On("DeleteAccount", mock.Anything, mock.Anything).Return(nil)
	svc, pins := newPinnedService(t, client)
	ctx := context.Background()

	require.NoError(t, svc.CreateUser(ctx, "alice", "password", strongKDF))
	got, ok, err := pins.Get(testServer, "alice")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, strongKDF, got)

	require.NoError(t, svc.DeleteAccount(ctx, "alice", "password"))
	_, ok, err = pins.Get(testServer, "alice")
	require.NoError(t, err)
	assert.False(t, ok, "логин освободился — параметры забыты")
}

// Временная учётка, созданная администратором для другого человека, не запоминается
// на устройстве администратора.
func TestAuthService_KDFPins_TemporaryUserNotRemembered(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	client := mocks.NewAuthClient(t)
	client.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, strongKDF, true).Return(&expires, nil)
	svc, pins := newPinnedService(t, client)

	_, _, err := svc.CreateTemporaryUser(context.Background(), "bob", strongKDF)
	require.NoError(t, err)

	_, ok, err := pins.Get(testServer, "bob")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestKDFParams_WeakerThan(t *testing.T) {
	base := domain.KDFParams{Time: 3, MemoryKiB: 64 * 1024, Threads: 4}
	assert.False(t, base.WeakerThan(base))
	assert.True(t, domain.KDFParams{Time: 2, MemoryKiB: 64 * 1024, Threads: 4}.WeakerThan(base))
	assert.True(t, domain.KDFParams{Time: 5, MemoryKiB: 32 * 1024, Threads: 4}.WeakerThan(base), "меньше памяти — слабее, даже при большем числе проходов")
	assert.False(t, domain.KDFParams{Time: 3, MemoryKiB: 64 * 1024, Threads: 1}.WeakerThan(base), "потоки не учитываются")
	assert.False(t, domain.KDFParams{Time: 4, MemoryKiB: 128 * 1024, Threads: 4}.WeakerThan(base))
}
