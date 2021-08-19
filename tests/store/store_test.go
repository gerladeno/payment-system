package store_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"sync"
	"testing"
)

type PgStoreSuite struct {
	pg     *pgStore.PG
	ctx    context.Context
	cancel context.CancelFunc
	suite.Suite
}

func (s *PgStoreSuite) SetupSuite() {
	log := &logrus.Logger{}
	var err error
	ctx := context.Background()
	s.pg, err = pgStore.GetPGStore(ctx, log, "postgresql://user:user_pw@localhost:5433/payments?sslmode=disable")
	require.NoError(s.T(), err)
	err = s.pg.Migrate(migrate.Up)
	require.NoError(s.T(), err)
}

func (s *PgStoreSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *PgStoreSuite) TearDownTest() {
	s.cancel()
	err := s.pg.Truncate()
	require.NoError(s.T(), err)
}

func (s *PgStoreSuite) TearDownSuite() {
	err := s.pg.Migrate(migrate.Down)
	require.NoError(s.T(), err)
	s.pg.DC()
}

func (s *PgStoreSuite) TestCreateWallets() {
	uid := uuid.New()
	err := s.pg.CreateWallet(s.ctx, uid.String(), 0)
	require.NoError(s.T(), err)
	err = s.pg.CreateWallet(s.ctx, uid.String(), 0)
	require.ErrorIs(s.T(), err, pkg.ErrDuplicateAction(uid.String()))
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uid := uuid.New()
			err := s.pg.CreateWallet(s.ctx, uid.String(), 0)
			require.NoError(s.T(), err)
		}()
	}
	wg.Wait()
}

func (s *PgStoreSuite) TestDepositWithdraw() {
	uid := uuid.New()
	err := s.pg.CreateWallet(s.ctx, uid.String(), 0)
	require.NoError(s.T(), err)
	err = s.pg.DepositWithdraw(s.ctx, uid.String(), 1000, "1")
	require.NoError(s.T(), err)
	err = s.pg.DepositWithdraw(s.ctx, uid.String(), 1000, "1")
	require.ErrorIs(s.T(), err, pkg.ErrDuplicateAction("1"))
	err = s.pg.DepositWithdraw(s.ctx, uid.String(), 1000, "2")
	require.NoError(s.T(), err)
	var wg sync.WaitGroup
	for i := 3; i < 103; i ++ {
		i:=i
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.pg.DepositWithdraw(s.ctx, uid.String(), -10, fmt.Sprintf("%d", i))
			require.NoError(s.T(), err)
		}()
	}
	err = s.pg.DepositWithdraw(s.ctx, uid.String(), -5000, "2000")
	require.ErrorIs(s.T(), err, pkg.ErrInsufficientFunds)
	wg.Wait()
	w, err := s.pg.GetWallet(s.ctx, uid.String())
	require.NoError(s.T(), err)
	require.Equal(s.T(), w.Amount, 1000.0)
	// Testing reports by types
	report, err := s.pg.Report(s.ctx, uid.String(), nil, nil, -1)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 102)
	report, err = s.pg.Report(s.ctx, uid.String(), nil, nil, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 2)
	report, err = s.pg.Report(s.ctx, uid.String(), nil, nil, 1)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 100)
	report, err = s.pg.Report(s.ctx, uid.String(), nil, nil, 2)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 0)
	report, err = s.pg.Report(s.ctx, uid.String(), nil, nil, 3)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 0)
}

func (s *PgStoreSuite) TestTransferFunds() {
	uid1 := uuid.New()
	err := s.pg.CreateWallet(s.ctx, uid1.String(), 0)
	require.NoError(s.T(), err)
	err = s.pg.DepositWithdraw(s.ctx, uid1.String(), 1000, "1")
	require.NoError(s.T(), err)
	uid2 := uuid.New()
	err = s.pg.CreateWallet(s.ctx, uid2.String(), 0)
	require.NoError(s.T(), err)
	err = s.pg.TransferFunds(s.ctx, uid1.String(), uid2.String(), 0.5, "2")
	require.NoError(s.T(), err)
	err = s.pg.TransferFunds(s.ctx, uid1.String(), uid2.String(), 0.5, "2")
	require.ErrorIs(s.T(), err, pkg.ErrDuplicateAction("2"))
	var wg sync.WaitGroup
	for i := 3; i < 103; i ++ {
		i:=i
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.pg.TransferFunds(s.ctx, uid1.String(), uid2.String(), 0.5, fmt.Sprintf("%d", i))
			require.NoError(s.T(), err)
		}()
	}
	err = s.pg.TransferFunds(s.ctx, uid1.String(), uid2.String(), 5000, "2000")
	require.ErrorIs(s.T(), err, pkg.ErrInsufficientFunds)
	wg.Wait()
	w, err := s.pg.GetWallet(s.ctx, uid1.String())
	require.NoError(s.T(), err)
	require.Equal(s.T(), w.Amount, 949.5)
	w, err = s.pg.GetWallet(s.ctx, uid2.String())
	require.NoError(s.T(), err)
	require.Equal(s.T(), w.Amount, 50.5)
	// Testing reports by types
	report, err := s.pg.Report(s.ctx, uid1.String(), nil, nil, -1)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 102)
	report, err = s.pg.Report(s.ctx, uid1.String(), nil, nil, 0)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 1)
	report, err = s.pg.Report(s.ctx, uid1.String(), nil, nil, 1)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 0)
	report, err = s.pg.Report(s.ctx, uid1.String(), nil, nil, 2)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 101)
	report, err = s.pg.Report(s.ctx, uid2.String(), nil, nil, 3)
	require.NoError(s.T(), err)
	require.Len(s.T(), report, 101)
}

func TestPgStoreSuite(t *testing.T) {
	// run ONLY on empty DB
	//s.T().Skip()
	suite.Run(t, new(PgStoreSuite))
}
