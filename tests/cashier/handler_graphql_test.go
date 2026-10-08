package cashier_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type CashierGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	userID     int
	merchantID int
	cashierID  int
}

func (s *CashierGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupCashierService chains the user and merchant services it depends on.
	s.SetupCashierService()

	s.userID = s.SeedUser(s.Ctx)
	s.merchantID = s.SeedMerchant(s.Ctx, s.userID)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

func (s *CashierGraphqlTestSuite) TestCashierGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateCashierRequest!) {
			createCashier(input: $input) { status message data { id merchant_id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"merchant_id": s.merchantID,
		"user_id":     s.userID,
		"name":        "Graphql Cashier",
	}})
	created := tests.Payload(s.T(), create, "createCashier")
	s.cashierID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.cashierID)
	s.Equal("Graphql Cashier", tests.Str(s.T(), created, "name"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdCashierRequest!) { findByIdCashier(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.cashierID}})
	s.Equal(s.cashierID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdCashier"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllCashier(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllCashier"))

	// 4. FindByActive
	active := s.gql.Do(`query { findByActiveCashier(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveCashier"))

	// 5. FindByMerchant
	byMerchant := s.gql.Do(`
		query ByMerchant($input: FindByMerchantCashierRequest!) { findByMerchantCashier(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"merchant_id": s.merchantID, "page": 1, "page_size": 10}})
	s.NotEmpty(tests.Items(s.T(), byMerchant, "findByMerchantCashier"))

	// 6. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateCashierRequest!) {
			updateCashier(input: $input) { status data { id name } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"cashier_id": s.cashierID,
		"name":       "Graphql Cashier Updated",
	}})
	s.Equal("Graphql Cashier Updated", tests.Str(s.T(), tests.Payload(s.T(), update, "updateCashier"), "name"))

	// 7. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdCashierRequest!) { trashedCashier(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.cashierID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedCashier"))

	trashedList := s.gql.Do(`query { findByTrashedCashier(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedCashier"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdCashierRequest!) { restoreCashier(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.cashierID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreCashier"))

	// 8. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdCashierRequest!) { trashedCashier(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.cashierID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdCashierRequest!) { deleteCashierPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.cashierID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteCashierPermanent"))

	// 9. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllCashier { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllCashier"))
	deleteAll := s.gql.Do(`mutation { deleteAllCashierPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllCashierPermanent"))
}

func TestCashierGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CashierGraphqlTestSuite))
}
