package transaction_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type TransactionGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	userID     int
	categoryID int
	merchantID int
	productID  int
	cashierID  int
	orderID    int
	transID    int
}

func (s *TransactionGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupTransactionService chains cashier, merchant, order and order-item.
	s.SetupTransactionService()

	s.userID = s.SeedUser(s.Ctx)
	s.categoryID = s.SeedCategory(s.Ctx)
	s.merchantID = s.SeedMerchant(s.Ctx, s.userID)
	s.productID = s.SeedProduct(s.Ctx, s.merchantID, s.categoryID)
	s.orderID = s.SeedOrder(s.Ctx, s.userID, s.merchantID, s.productID)
	s.cashierID = s.cashierIDFor(s.userID, s.merchantID)

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

// cashierIDFor reads the cashier SeedOrder created for the user/merchant pair.
func (s *TransactionGraphqlTestSuite) cashierIDFor(userID, merchantID int) int {
	var id int
	err := s.SQLxDB().QueryRowContext(s.Ctx,
		`SELECT cashier_id FROM cashiers WHERE user_id = $1 AND merchant_id = $2 AND deleted_at IS NULL LIMIT 1`,
		userID, merchantID).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *TransactionGraphqlTestSuite) TestTransactionGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateTransactionInput!) {
			createTransaction(input: $input) {
				status message data { id orderId merchantId paymentMethod amount paymentStatus }
			}
		}`, map[string]interface{}{"input": map[string]interface{}{
		"orderId":       s.orderID,
		"cashierId":     s.cashierID,
		"paymentMethod": "E-Wallet",
		"amount":        100000,
		"paymentStatus": "pending",
	}})
	created := tests.Payload(s.T(), create, "createTransaction")
	s.transID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.transID)
	s.Equal("E-Wallet", tests.Str(s.T(), created, "paymentMethod"))

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdTransactionInput!) {
			findByIdTransaction(input: $input) { status data { id paymentMethod } }
		}`, map[string]interface{}{"input": map[string]interface{}{"id": s.transID}})
	s.Equal(s.transID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdTransaction"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllTransaction(input: {page: 1, pageSize: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllTransaction"))

	// 4. FindByMerchant
	byMerchant := s.gql.Do(`
		query ByMerchant($input: FindAllTransactionMerchantInput!) {
			findByMerchantTransaction(input: $input) { status data { id } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"merchantId": s.merchantID, "page": 1, "pageSize": 10,
	}})
	s.NotEmpty(tests.Items(s.T(), byMerchant, "findByMerchantTransaction"))

	// 5. FindByActive
	active := s.gql.Do(`query { findByActiveTransaction(input: {page: 1, pageSize: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveTransaction"))

	// 6. Update
	update := s.gql.Do(`
		mutation Update($input: UpdateTransactionInput!) {
			updateTransaction(input: $input) { status data { id paymentMethod amount } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"transactionId": s.transID,
		"orderId":       s.orderID,
		"cashierId":     s.cashierID,
		"paymentMethod": "Credit Card",
		"amount":        150000,
		"paymentStatus": "success",
	}})
	s.Equal("Credit Card", tests.Str(s.T(), tests.Payload(s.T(), update, "updateTransaction"), "paymentMethod"))

	// 7. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdTransactionInput!) { trashedTransaction(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.transID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedTransaction"))

	trashedList := s.gql.Do(`query { findByTrashedTransaction(input: {page: 1, pageSize: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedTransaction"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdTransactionInput!) { restoreTransaction(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.transID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreTransaction"))

	// 8. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdTransactionInput!) { trashedTransaction(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.transID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdTransactionInput!) {
			deleteTransactionPermanent(input: $input) { status }
		}`, map[string]interface{}{"input": map[string]interface{}{"id": s.transID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteTransactionPermanent"))

	// 9. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllTransaction { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllTransaction"))
	deleteAll := s.gql.Do(`mutation { deleteAllTransactionPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllTransactionPermanent"))
}

func TestTransactionGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionGraphqlTestSuite))
}
