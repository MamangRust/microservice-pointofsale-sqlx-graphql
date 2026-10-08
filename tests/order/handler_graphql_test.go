package order_test

import (
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type OrderGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql        *tests.GraphQLClient
	userID     int
	categoryID int
	merchantID int
	productID  int
	cashierID  int
	orderID    int
}

func (s *OrderGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupOrderService chains cashier, merchant, product and order-item.
	s.SetupOrderService()

	s.userID = s.SeedUser(s.Ctx)
	s.categoryID = s.SeedCategory(s.Ctx)
	s.merchantID = s.SeedMerchant(s.Ctx, s.userID)
	s.productID = s.SeedProduct(s.Ctx, s.merchantID, s.categoryID)
	s.cashierID = s.seedCashier()

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

// seedCashier inserts a cashier row directly: the cashier service is running,
// but seeding through SQL keeps the order suite independent of its lifecycle.
func (s *OrderGraphqlTestSuite) seedCashier() int {
	var id int
	err := s.SQLxDB().QueryRowContext(s.Ctx, `
		INSERT INTO cashiers (merchant_id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, 'Graphql Cashier', current_timestamp, current_timestamp)
		RETURNING cashier_id
	`, s.merchantID, s.userID).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *OrderGraphqlTestSuite) TestOrderGraphqlLifecycle() {
	// 1. Create
	create := s.gql.Do(`
		mutation Create($input: CreateOrderInput!) {
			createOrder(input: $input) { status message data { id total_price } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"merchant_id": s.merchantID,
		"cashier_id":  s.cashierID,
		"items": []map[string]interface{}{
			{"product_id": s.productID, "quantity": 1},
		},
	}})
	created := tests.Payload(s.T(), create, "createOrder")
	s.orderID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.orderID)

	// 2. FindById
	found := s.gql.Do(`
		query Find($input: FindByIdOrderInput!) { findByIdOrder(input: $input) { status data { id } } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal(s.orderID, tests.Int(s.T(), tests.Payload(s.T(), found, "findByIdOrder"), "id"))

	// 3. FindAll
	all := s.gql.Do(`query { findAllOrder(input: {page: 1, page_size: 10}) { status data { id } } }`, nil)
	s.NotEmpty(tests.Items(s.T(), all, "findAllOrder"))

	// 4. FindByActive
	active := s.gql.Do(`query { findByActiveOrder(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), active, "findByActiveOrder"))

	// 5. Update (re-uses the order item created with the order)
	orderItemID := s.orderItemID(s.orderID)
	update := s.gql.Do(`
		mutation Update($input: UpdateOrderInput!) {
			updateOrder(input: $input) { status data { id } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"order_id": s.orderID,
		"items": []map[string]interface{}{
			{"order_item_id": orderItemID, "product_id": s.productID, "quantity": 2},
		},
	}})
	s.Equal(s.orderID, tests.Int(s.T(), tests.Payload(s.T(), update, "updateOrder"), "id"))

	// 6. Trash / findByTrashed / restore
	trashed := s.gql.Do(`
		mutation Trash($input: FindByIdOrderInput!) { trashedOrder(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", tests.Status(s.T(), trashed, "trashedOrder"))

	trashedList := s.gql.Do(`query { findByTrashedOrder(input: {page: 1, page_size: 10}) { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), trashedList, "findByTrashedOrder"))

	restored := s.gql.Do(`
		mutation Restore($input: FindByIdOrderInput!) { restoreOrder(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", tests.Status(s.T(), restored, "restoreOrder"))

	// 7. DeletePermanent
	s.gql.Do(`
		mutation Trash($input: FindByIdOrderInput!) { trashedOrder(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	permanent := s.gql.Do(`
		mutation DeletePermanent($input: FindByIdOrderInput!) { deleteOrderPermanent(input: $input) { status } }`,
		map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", tests.Status(s.T(), permanent, "deleteOrderPermanent"))

	// 8. Bulk
	restoreAll := s.gql.Do(`mutation { restoreAllOrder { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), restoreAll, "restoreAllOrder"))
	deleteAll := s.gql.Do(`mutation { deleteAllOrderPermanent { status } }`, nil)
	s.Equal("success", tests.Status(s.T(), deleteAll, "deleteAllOrderPermanent"))
}

func (s *OrderGraphqlTestSuite) orderItemID(orderID int) int {
	var id int
	err := s.SQLxDB().QueryRowContext(s.Ctx,
		`SELECT order_item_id FROM order_items WHERE order_id = $1 ORDER BY order_item_id LIMIT 1`, orderID).Scan(&id)
	s.Require().NoError(err)
	return id
}

func TestOrderGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(OrderGraphqlTestSuite))
}
